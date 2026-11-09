package project

import (
	"context"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/id"
	sdkproject "github.com/appwrite/sdk-for-go/v7/project"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

var (
	_ resource.Resource                   = &platformResource{}
	_ resource.ResourceWithConfigure      = &platformResource{}
	_ resource.ResourceWithImportState    = &platformResource{}
	_ resource.ResourceWithValidateConfig = &platformResource{}
)

type platformResource struct {
	clients *common.AppwriteClients
}

type platformResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Type                  types.String `tfsdk:"type"`
	Name                  types.String `tfsdk:"name"`
	Hostname              types.String `tfsdk:"hostname"`
	ApplicationID         types.String `tfsdk:"application_id"`
	BundleIdentifier      types.String `tfsdk:"bundle_identifier"`
	PackageName           types.String `tfsdk:"package_name"`
	PackageIdentifierName types.String `tfsdk:"package_identifier_name"`
	CreatedAt             types.String `tfsdk:"created_at"`
	UpdatedAt             types.String `tfsdk:"updated_at"`
	ProjectID             types.String `tfsdk:"project_id"`
}

func NewPlatformResource() resource.Resource {
	return &platformResource{}
}

func (r *platformResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_platform"
}

func (r *platformResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	identifier := func(description, platformType string, maxLength int) schema.StringAttribute {
		return schema.StringAttribute{
			Description: fmt.Sprintf("%s Required for %s platforms and not accepted for other types. "+
				"At most %d characters.", description, platformType, maxLength),
			Optional:   true,
			Validators: []validator.String{textValidator{maxLength: maxLength}},
		}
	}

	resp.Schema = schema.Schema{
		Description: "Manages a platform of an Appwrite project: a web app, or an Android, Apple, Linux or " +
			"Windows app, allowed to call the project's APIs from a client SDK.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The platform ID. Generated when not set. Platform IDs are unique across the whole " +
					"Appwrite instance, not only the project, so a custom ID another project uses is rejected. " +
					common.ForcesReplacementNote,
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, maxPlatformIDLength),
					stringvalidator.RegexMatches(platformIDPattern,
						"must contain only letters, digits, periods, hyphens and underscores, and must not start with a period, hyphen or underscore"),
				},
			},
			"type": schema.StringAttribute{
				Description:   "The platform type: " + strings.Join(platformTypeNames(), ", ") + ". " + common.ForcesReplacementNote,
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.OneOf(platformTypeNames()...)},
			},
			"name": schema.StringAttribute{
				Description: fmt.Sprintf("The platform name. At most %d characters.", maxPlatformNameLength),
				Required:    true,
				Validators:  []validator.String{textValidator{maxLength: maxPlatformNameLength}},
			},
			"hostname": func() schema.StringAttribute {
				attribute := identifier("The host name the web app is served from, such as app.example.com, "+
					"without a scheme, port or path. A leading wildcard such as *.example.com allows every "+
					"subdomain, though not example.com itself.", platformTypeWeb, 253)
				attribute.Validators = append(attribute.Validators,
					stringvalidator.RegexMatches(hostnamePattern, "must be a host name without a scheme, port or path"))
				return attribute
			}(),
			"application_id": identifier("The Android application ID, such as com.example.app.", platformTypeAndroid, 255),
			"bundle_identifier": identifier("The Apple bundle identifier, such as com.example.app.",
				platformTypeApple, 255),
			"package_name":            identifier("The Linux package name.", platformTypeLinux, 255),
			"package_identifier_name": identifier("The Windows package identifier name.", platformTypeWindows, 255),
			"created_at": schema.StringAttribute{
				Description:   "The platform creation timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Description: "The platform last update timestamp in ISO 8601 format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					common.UseStateForUnknownUnlessUpdating(),
				},
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

// ValidateConfig catches at plan time what the API would otherwise reject
// mid-apply: the identifying argument of the chosen type left out, or another
// type's given instead.
func (r *platformResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config platformResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.Type.IsUnknown() || config.Type.IsNull() {
		return
	}
	resp.Diagnostics.Append(validatePlatformConfig(config)...)
}

func (r *platformResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *platformResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan platformResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, client, ok := r.client(plan.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	platformID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		platformID = id.Unique()
	}

	platform, err := createPlatform(client, platformID, plan)
	if err != nil {
		resp.Diagnostics.AddError("Error creating platform", platformErrorDetail(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	if err := mapPlatformToState(platform, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading created platform", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *platformResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state platformResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, client, ok := r.client(state.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	platform, err := client.GetPlatform(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading platform", platformErrorDetail(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	if err := mapPlatformToState(platform, &state); err != nil {
		resp.Diagnostics.AddError("Error reading platform", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *platformResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state platformResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, client, ok := r.client(plan.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	plan.ID = state.ID
	platform, err := updatePlatform(client, plan)
	if err != nil {
		resp.Diagnostics.AddError("Error updating platform", platformErrorDetail(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	if err := mapPlatformToState(platform, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading updated platform", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *platformResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state platformResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, client, ok := r.client(state.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	_, err := client.DeletePlatform(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting platform", platformErrorDetail(err))
	}
}

func (r *platformResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectID, platformID, err := platformImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	if projectID != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectID)...)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), platformID)...)
}

func (r *platformResource) client(projectID types.String, diags *diag.Diagnostics) (string, *sdkproject.Project, bool) {
	resolved, err := common.ResolveProjectID(r.clients, projectID)
	if err != nil {
		diags.AddError("Error resolving project ID", err.Error())
		return "", nil, false
	}
	if err := common.ValidateProjectCredential(r.clients, platformResourceName, platformScopes...); err != nil {
		diags.AddError("Unsupported Appwrite credential", err.Error())
		return "", nil, false
	}
	return resolved, appwrite.NewProject(r.clients.ClientForProject(resolved)), true
}
