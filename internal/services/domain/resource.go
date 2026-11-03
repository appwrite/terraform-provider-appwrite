package domain

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

const domainResourceName = "appwrite_domain"

var (
	_ resource.Resource                = &domainResource{}
	_ resource.ResourceWithConfigure   = &domainResource{}
	_ resource.ResourceWithImportState = &domainResource{}
)

type domainResource struct {
	clients *common.AppwriteClients
}

type domainResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Domain              types.String `tfsdk:"domain"`
	OrganizationID      types.String `tfsdk:"organization_id"`
	Registrar           types.String `tfsdk:"registrar"`
	NameserversVerified types.Bool   `tfsdk:"nameservers_verified"`
	AppwriteNameservers types.List   `tfsdk:"appwrite_nameservers"`
	AutoRenewal         types.Bool   `tfsdk:"auto_renewal"`
	ExpiresAt           types.String `tfsdk:"expires_at"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

func NewDomainResource() resource.Resource {
	return &domainResource{}
}

func (r *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	useState := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	resp.Schema = schema.Schema{
		Description: "Adds a domain to an Appwrite organization, so Appwrite hosts its DNS zone.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   domainIDDescription,
				Computed:      true,
				PlanModifiers: useState,
			},
			"domain": schema.StringAttribute{
				Description: "The domain name, such as example.com. Must be lowercase. " +
					common.ForcesReplacementNote,
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    domainNameValidators(),
			},
			"organization_id": common.OrganizationIDAttribute(),
			"registrar": schema.StringAttribute{
				Description:   registrarDescription,
				Computed:      true,
				PlanModifiers: useState,
			},
			"nameservers_verified": schema.BoolAttribute{
				Description: "Whether Appwrite found the domain delegated to its nameservers when it last checked. " +
					"Appwrite checks when the domain is added and when its nameservers are updated or verified, and " +
					"not otherwise, so a change made at the registrar shows here only after " +
					"appwrite_domain_nameservers_verification or the Console runs a check.",
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"appwrite_nameservers": schema.ListAttribute{
				Description: "The nameservers Appwrite serves this domain's zone from. Point the domain's " +
					"registrar at these to delegate DNS to Appwrite.",
				ElementType:   types.StringType,
				Computed:      true,
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"auto_renewal": schema.BoolAttribute{
				Description:   "Whether a domain registered through Appwrite renews automatically.",
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"expires_at": schema.StringAttribute{
				Description:   "When a domain registered through Appwrite expires, in ISO 8601 format. Empty for a domain registered elsewhere.",
				Computed:      true,
				PlanModifiers: useState,
			},
			"created_at": schema.StringAttribute{
				Description:   "The domain creation timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: useState,
			},
			"updated_at": schema.StringAttribute{
				Description:   "The domain last update timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: useState,
			},
		},
	}
}

func (r *domainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := common.ResolveOrganizationID(r.clients, plan.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}
	client, err := domainsClient(r.clients, domainResourceName, organizationID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	// An organization is a team in the console project, so its ID is the
	// team ID the API asks for.
	created, err := client.Create(organizationID, plan.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating domain", common.FormatErrorWithAuthGuidance(err, credentialGuidance(domainResourceName)))
		return
	}

	// Read it back so the first apply sees the same records a refresh does:
	// appwrite_nameservers is derived from them, and a registrar resource
	// wired to it should not get a different list on the first run.
	created, err = client.Get(created.Id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created domain", common.FormatErrorWithAuthGuidance(err, credentialGuidance(domainResourceName)))
		return
	}

	plan.OrganizationID = types.StringValue(organizationID)
	resp.Diagnostics.Append(mapDomainToState(ctx, created, &plan, true)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := common.ResolveOrganizationID(r.clients, state.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}
	client, err := domainsClient(r.clients, domainResourceName, organizationID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	found, err := client.Get(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading domain", common.FormatErrorWithAuthGuidance(err, credentialGuidance(domainResourceName)))
		return
	}

	state.OrganizationID = types.StringValue(organizationID)
	resp.Diagnostics.Append(mapDomainToState(ctx, found, &state, true)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is unreachable: every argument forces replacement. The framework still
// requires the method.
func (r *domainResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Domains cannot be updated in place. Change the domain to replace it.")
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := common.ResolveOrganizationID(r.clients, state.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}
	client, err := domainsClient(r.clients, domainResourceName, organizationID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	_, err = client.Delete(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting domain", common.FormatErrorWithAuthGuidance(err, credentialGuidance(domainResourceName)))
	}
}

func (r *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	switch len(parts) {
	case 1:
		if parts[0] == "" {
			resp.Diagnostics.AddError("Invalid import ID", "Expected domain_id or organization_id/domain_id")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
	case 2:
		if parts[0] == "" || parts[1] == "" {
			resp.Diagnostics.AddError("Invalid import ID", "Expected domain_id or organization_id/domain_id")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected domain_id or organization_id/domain_id, got: %s", req.ID))
	}
}
