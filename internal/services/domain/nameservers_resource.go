package domain

import (
	"context"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/domains"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

const nameserversResourceName = "appwrite_domain_nameservers"

// The API accepts at most 13 nameservers, the most a DNS referral can carry.
const maxNameservers = 13

var (
	_ resource.Resource                = &nameserversResource{}
	_ resource.ResourceWithConfigure   = &nameserversResource{}
	_ resource.ResourceWithImportState = &nameserversResource{}
)

type nameserversResource struct {
	clients *common.AppwriteClients
}

type nameserversResourceModel struct {
	ID             types.String `tfsdk:"id"`
	DomainID       types.String `tfsdk:"domain_id"`
	Nameservers    types.List   `tfsdk:"nameservers"`
	OrganizationID types.String `tfsdk:"organization_id"`
}

func NewNameserversResource() resource.Resource {
	return &nameserversResource{}
}

func (r *nameserversResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_nameservers"
}

func (r *nameserversResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Sets the registrar nameservers of a domain registered through Appwrite. Domains registered " +
			"elsewhere have their nameservers set at that registrar; use appwrite_domain_nameservers_verification " +
			"for those instead.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   domainIDDescription,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"domain_id": schema.StringAttribute{
				Description: "The ID of a domain registered through Appwrite. " +
					common.ForcesReplacementNote,
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"nameservers": schema.ListAttribute{
				Description: fmt.Sprintf("The nameservers to delegate the domain to, at most %d. Omit to "+
					"delegate it to Appwrite's nameservers, which is what lets Appwrite serve its DNS records. "+
					"Appwrite does not report the nameservers it last set, so a change made outside Terraform "+
					"is not detected.", maxNameservers),
				ElementType: types.StringType,
				Optional:    true,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, maxNameservers),
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(
						stringvalidator.LengthBetween(1, 253),
						stringvalidator.RegexMatches(nameserverPattern, "must be a host name such as ns1.example.net"),
					),
				},
			},
			"organization_id": common.OrganizationIDAttribute(),
		},
	}
}

func (r *nameserversResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData))
		return
	}
	r.clients = clients
}

func (r *nameserversResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan nameserversResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &plan, &resp.State, &resp.Diagnostics)
}

func (r *nameserversResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan nameserversResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &plan, &resp.State, &resp.Diagnostics)
}

// apply sets the registrar nameservers. Create and update are the same call:
// the API replaces the whole list, and an empty request means Appwrite's own.
func (r *nameserversResource) apply(ctx context.Context, plan *nameserversResourceModel, state *tfsdk.State, diags *diag.Diagnostics) {
	organizationID, err := common.ResolveOrganizationID(r.clients, plan.OrganizationID)
	if err != nil {
		diags.AddError("Error resolving organization ID", err.Error())
		return
	}
	client, err := domainsClient(r.clients, nameserversResourceName, organizationID)
	if err != nil {
		diags.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	var opts []domains.UpdateNameserversOption
	if !plan.Nameservers.IsNull() {
		var nameservers []string
		diags.Append(plan.Nameservers.ElementsAs(ctx, &nameservers, false)...)
		if diags.HasError() {
			return
		}
		opts = append(opts, client.WithUpdateNameserversNameservers(nameservers))
	}

	updated, err := client.UpdateNameservers(plan.DomainID.ValueString(), opts...)
	if err != nil {
		diags.AddError("Error updating domain nameservers", common.FormatErrorWithAuthGuidance(err, credentialGuidance(nameserversResourceName)))
		return
	}

	plan.ID = types.StringValue(updated.Id)
	plan.OrganizationID = types.StringValue(organizationID)
	diags.Append(state.Set(ctx, plan)...)
}

func (r *nameserversResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state nameserversResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := common.ResolveOrganizationID(r.clients, state.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}
	client, err := domainsClient(r.clients, nameserversResourceName, organizationID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	// The API does not return the registrar nameservers, so the configured
	// list is kept as it is. All a read can establish is that the domain still
	// exists.
	domain, err := client.Get(state.DomainID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading domain", common.FormatErrorWithAuthGuidance(err, credentialGuidance(nameserversResourceName)))
		return
	}

	state.ID = types.StringValue(domain.Id)
	state.OrganizationID = types.StringValue(organizationID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete leaves the registrar nameservers as they are and only stops managing
// them. Resetting them on destroy would move the domain's DNS to another
// provider as a side effect of removing a resource, which is not something a
// destroy should do silently.
func (r *nameserversResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *nameserversResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	switch {
	case len(parts) == 1 && parts[0] != "":
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	case len(parts) == 2 && parts[0] != "" && parts[1] != "":
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[1])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected domain_id or organization_id/domain_id, got: %s", req.ID))
	}
}
