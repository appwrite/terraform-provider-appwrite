package project

import (
	"context"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v6/appwrite"
	"github.com/appwrite/sdk-for-go/v6/id"
	"github.com/appwrite/sdk-for-go/v6/models"
	sdkproject "github.com/appwrite/sdk-for-go/v6/project"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &keyResource{}
	_ resource.ResourceWithConfigure   = &keyResource{}
	_ resource.ResourceWithImportState = &keyResource{}
)

type keyResource struct {
	clients *common.AppwriteClients
}

type keyResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Scopes         types.Set    `tfsdk:"scopes"`
	Expire         types.String `tfsdk:"expire"`
	Secret         types.String `tfsdk:"secret"`
	AccessedAt     types.String `tfsdk:"accessed_at"`
	SDKs           types.Set    `tfsdk:"sdks"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	ProjectID      types.String `tfsdk:"project_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
}

func NewKeyResource() resource.Resource {
	return &keyResource{}
}

func (r *keyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_key"
}

func (r *keyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite project API key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The API key ID.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The API key name.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1), stringvalidator.LengthAtMost(128)},
			},
			"scopes": schema.SetAttribute{
				Description: "The permission scopes granted to the API key. A maximum of 100 scopes is allowed.",
				Required:    true,
				ElementType: types.StringType,
				Validators:  []validator.Set{setvalidator.SizeAtMost(100)},
			},
			"expire": schema.StringAttribute{
				Description: "The expiration timestamp in ISO 8601 format. Omit for no expiration.",
				Optional:    true,
			},
			"secret": schema.StringAttribute{
				Description: "The API key secret.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"accessed_at": schema.StringAttribute{
				Description: "The most recent access timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"sdks": schema.SetAttribute{
				Description: "The SDKs that have used this API key.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.StringAttribute{
				Description: "The API key creation timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The API key last update timestamp in ISO 8601 format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					common.UseStateForUnknownUnlessUpdating(),
				},
			},
			"project_id":      common.ProjectIDAttribute(),
			"organization_id": common.OrganizationIDAttribute(),
		},
	}
}

func (r *keyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *keyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan keyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	if err := common.ValidateOrganizationCredential(r.clients, "appwrite_project_key", "keys.read", "keys.write"); err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}
	projectClient, organizationID, err := r.client(projectID, plan.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}

	keyID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		keyID = id.Unique()
	}

	var scopes []string
	resp.Diagnostics.Append(plan.Scopes.ElementsAs(ctx, &scopes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var opts []sdkproject.CreateKeyOption
	if !plan.Expire.IsNull() && !plan.Expire.IsUnknown() {
		opts = append(opts, projectClient.WithCreateKeyExpire(plan.Expire.ValueString()))
	}

	key, err := projectClient.CreateKey(keyID, plan.Name.ValueString(), scopes, opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error creating project API key", common.FormatErrorWithAuthGuidance(err, common.OrganizationCredentialGuidance("appwrite_project_key", "keys.read", "keys.write")))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	plan.OrganizationID = types.StringValue(organizationID)
	r.mapToState(ctx, key, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *keyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state keyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	if err := common.ValidateOrganizationCredential(r.clients, "appwrite_project_key", "keys.read", "keys.write"); err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}
	projectClient, organizationID, err := r.client(projectID, state.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}

	key, err := projectClient.GetKey(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading project API key", common.FormatErrorWithAuthGuidance(err, common.OrganizationCredentialGuidance("appwrite_project_key", "keys.read", "keys.write")))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	state.OrganizationID = types.StringValue(organizationID)
	r.mapToState(ctx, key, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *keyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan keyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	if err := common.ValidateOrganizationCredential(r.clients, "appwrite_project_key", "keys.read", "keys.write"); err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}
	projectClient, organizationID, err := r.client(projectID, plan.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}

	var scopes []string
	resp.Diagnostics.Append(plan.Scopes.ElementsAs(ctx, &scopes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var opts []sdkproject.UpdateKeyOption
	if !plan.Expire.IsNull() && !plan.Expire.IsUnknown() {
		opts = append(opts, projectClient.WithUpdateKeyExpire(plan.Expire.ValueString()))
	}

	key, err := projectClient.UpdateKey(plan.ID.ValueString(), plan.Name.ValueString(), scopes, opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error updating project API key", common.FormatErrorWithAuthGuidance(err, common.OrganizationCredentialGuidance("appwrite_project_key", "keys.read", "keys.write")))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	plan.OrganizationID = types.StringValue(organizationID)
	r.mapToState(ctx, key, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *keyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state keyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	if err := common.ValidateOrganizationCredential(r.clients, "appwrite_project_key", "keys.read", "keys.write"); err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}
	projectClient, _, err := r.client(projectID, state.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}

	_, err = projectClient.DeleteKey(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting project API key", common.FormatErrorWithAuthGuidance(err, common.OrganizationCredentialGuidance("appwrite_project_key", "keys.read", "keys.write")))
	}
}

func (r *keyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	for _, part := range parts {
		if part == "" {
			resp.Diagnostics.AddError("Invalid import ID", "Expected key_id, project_id/key_id, or organization_id/project_id/key_id")
			return
		}
	}
	switch len(parts) {
	case 1:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
	case 2:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	case 3:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[1])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected key_id, project_id/key_id, or organization_id/project_id/key_id, got: %s", req.ID))
	}
}

func (r *keyResource) client(projectID string, resourceOrganizationID types.String) (*sdkproject.Project, string, error) {
	organizationID, err := common.ResolveOrganizationID(r.clients, resourceOrganizationID)
	if err != nil {
		return nil, "", err
	}
	client := r.clients.ClientForOrganizationProject(projectID, organizationID)
	return appwrite.NewProject(client), organizationID, nil
}

func (r *keyResource) mapToState(ctx context.Context, key *models.Key, model *keyResourceModel, diagnostics *diag.Diagnostics) {
	model.ID = types.StringValue(key.Id)
	model.Name = types.StringValue(key.Name)
	model.CreatedAt = types.StringValue(key.CreatedAt)
	model.UpdatedAt = types.StringValue(key.UpdatedAt)

	if key.Expire == "" {
		model.Expire = types.StringNull()
	} else {
		model.Expire = types.StringValue(key.Expire)
	}
	if key.Secret != "" {
		model.Secret = types.StringValue(key.Secret)
	}
	if key.AccessedAt == "" {
		model.AccessedAt = types.StringNull()
	} else {
		model.AccessedAt = types.StringValue(key.AccessedAt)
	}

	scopes, diags := types.SetValueFrom(ctx, types.StringType, key.Scopes)
	diagnostics.Append(diags...)
	model.Scopes = scopes

	sdks, diags := types.SetValueFrom(ctx, types.StringType, key.Sdks)
	diagnostics.Append(diags...)
	model.SDKs = sdks
}
