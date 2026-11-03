package domain

import (
	"context"
	"crypto/md5" //nolint:gosec // Appwrite derives domain IDs from MD5; this is not a security use
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/domains"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/query"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

var (
	_ datasource.DataSource                     = &domainDataSource{}
	_ datasource.DataSourceWithConfigure        = &domainDataSource{}
	_ datasource.DataSourceWithConfigValidators = &domainDataSource{}
)

type domainDataSource struct {
	clients *common.AppwriteClients
}

func NewDomainDataSource() datasource.DataSource {
	return &domainDataSource{}
}

func (d *domainDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *domainDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a domain in an Appwrite organization by ID or by name. Use it to manage the DNS " +
			"records of a domain that Terraform does not own, such as one bought through the Console.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The domain ID. Exactly one of id or domain must be set.",
				Optional:    true,
				Computed:    true,
			},
			"domain": schema.StringAttribute{
				Description: "The domain name, such as example.com. Exactly one of id or domain must be set.",
				Optional:    true,
				Computed:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: common.OrganizationIDDescription,
				Optional:    true,
				Computed:    true,
			},
			"registrar": schema.StringAttribute{
				Description: registrarDescription,
				Computed:    true,
			},
			"nameservers_verified": schema.BoolAttribute{
				Description: "Whether Appwrite found the domain delegated to its nameservers when it last checked.",
				Computed:    true,
			},
			"appwrite_nameservers": schema.ListAttribute{
				Description: "The nameservers Appwrite serves this domain's zone from.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"auto_renewal": schema.BoolAttribute{
				Description: "Whether a domain registered through Appwrite renews automatically.",
				Computed:    true,
			},
			"expires_at": schema.StringAttribute{
				Description: "When a domain registered through Appwrite expires, in ISO 8601 format. Empty for a domain registered elsewhere.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The domain creation timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The domain last update timestamp in ISO 8601 format.",
				Computed:    true,
			},
		},
	}
}

func (d *domainDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("domain")),
	}
}

func (d *domainDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData),
		)
		return
	}
	d.clients = clients
}

func (d *domainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config domainResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := common.ResolveOrganizationID(d.clients, config.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}
	client, err := domainsClientWithScopes(d.clients, domainResourceName, organizationID, dataSourceScopes...)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	var found *models.Domain
	if !config.ID.IsNull() {
		found, err = client.Get(config.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading domain", common.FormatErrorWithAuthGuidance(err, dataSourceGuidance))
			return
		}
	} else {
		found, err = findDomainByName(client, organizationID, config.Domain.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading domain", common.FormatErrorWithAuthGuidance(err, dataSourceGuidance))
			return
		}
		if found == nil {
			resp.Diagnostics.AddError("Domain not found", fmt.Sprintf("No domain named %q exists in organization %q.", config.Domain.ValueString(), organizationID))
			return
		}
	}

	config.OrganizationID = types.StringValue(organizationID)
	resp.Diagnostics.Append(mapDomainToState(ctx, found, &config, false)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// The data source only reads, so it needs only the read scope.
var (
	dataSourceScopes   = []string{"domains.read"}
	dataSourceGuidance = common.OrganizationCredentialGuidance(domainResourceName+" data source", dataSourceScopes...)
)

// findDomainByName returns the named domain, or nil when the organization has
// none by that name.
//
// A domain added through the API has the MD5 of its lowercase name as its ID,
// so that is tried first and usually settles it in one request. Domains bought
// or transferred through Appwrite hash the name as it was written instead, so a
// miss falls back to a query on the name.
func findDomainByName(client *domains.Domains, organizationID, name string) (*models.Domain, error) {
	sum := md5.Sum([]byte(strings.ToLower(name)))
	domain, err := client.Get(hex.EncodeToString(sum[:]))
	switch {
	case err == nil && strings.EqualFold(domain.Domain, name):
		return domain, nil
	case err != nil && !common.IsNotFoundError(err):
		return nil, err
	}

	names := []string{name}
	if lower := strings.ToLower(name); lower != name {
		names = append(names, lower)
	}
	list, err := client.List(client.WithListQueries([]string{
		query.Equal("teamId", organizationID),
		query.Equal("domain", names),
		query.Limit(1),
	}))
	if err != nil {
		return nil, err
	}
	if len(list.Domains) == 0 {
		return nil, nil
	}
	// Read it back by ID: a list is not guaranteed to carry each domain's DNS
	// records, and appwrite_nameservers is derived from them.
	return client.Get(list.Domains[0].Id)
}
