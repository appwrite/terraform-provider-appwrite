package site

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/appwrite/sdk-for-go/v2/appwrite"
	appwritefile "github.com/appwrite/sdk-for-go/v2/file"
	"github.com/appwrite/sdk-for-go/v2/models"
	"github.com/appwrite/sdk-for-go/v2/sites"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &deploymentResource{}
	_ resource.ResourceWithConfigure   = &deploymentResource{}
	_ resource.ResourceWithImportState = &deploymentResource{}
)

type deploymentResource struct {
	clients *common.AppwriteClients
}

type deploymentResourceModel struct {
	ID              types.String `tfsdk:"id"`
	SiteID          types.String `tfsdk:"site_id"`
	SourceType      types.String `tfsdk:"source_type"`
	Activate        types.Bool   `tfsdk:"activate"`
	WaitForReady    types.Bool   `tfsdk:"wait_for_ready"`
	CodePath        types.String `tfsdk:"code_path"`
	CodeHash        types.String `tfsdk:"code_hash"`
	InstallCommand  types.String `tfsdk:"install_command"`
	BuildCommand    types.String `tfsdk:"build_command"`
	OutputDirectory types.String `tfsdk:"output_directory"`
	Type            types.String `tfsdk:"type"`
	Reference       types.String `tfsdk:"reference"`
	Repository      types.String `tfsdk:"repository"`
	Owner           types.String `tfsdk:"owner"`
	RootDirectory   types.String `tfsdk:"root_directory"`
	Status          types.String `tfsdk:"status"`
	BuildLogs       types.String `tfsdk:"build_logs"`
	BuildDuration   types.Int64  `tfsdk:"build_duration"`
	SourceSize      types.Int64  `tfsdk:"source_size"`
	BuildSize       types.Int64  `tfsdk:"build_size"`
	TotalSize       types.Int64  `tfsdk:"total_size"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
	ProjectID       types.String `tfsdk:"project_id"`
}

func NewDeploymentResource() resource.Resource {
	return &deploymentResource{}
}

func (r *deploymentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_deployment"
}

func (r *deploymentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite site deployment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The deployment ID.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"site_id": schema.StringAttribute{
				Description:   "The site ID this deployment belongs to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"source_type": schema.StringAttribute{
				Description:   `The deployment source type. Must be one of "code", "vcs", or "template".`,
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"activate": schema.BoolAttribute{
				Description: "Whether to activate this deployment after creation.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"wait_for_ready": schema.BoolAttribute{
				Description: "Whether to wait for the deployment to reach ready status before completing. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"code_path": schema.StringAttribute{
				Description:   "Local path to the code file to upload. Required when source_type is code.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"code_hash": schema.StringAttribute{
				Description:   "Hash of the code file for drift detection. Use filesha256() to compute.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"install_command": schema.StringAttribute{
				Description:   "Custom install command for code deployments.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"build_command": schema.StringAttribute{
				Description:   "Custom build command for code deployments.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"output_directory": schema.StringAttribute{
				Description:   "Build output directory for code deployments.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				Description:   `Reference type for VCS and template deployments (e.g. "branch", "tag", "commit").`,
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"reference": schema.StringAttribute{
				Description:   "Reference value for VCS and template deployments (e.g. branch name, tag, or commit hash).",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"repository": schema.StringAttribute{
				Description:   "Repository name for template deployments.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"owner": schema.StringAttribute{
				Description:   "Repository owner for template deployments.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"root_directory": schema.StringAttribute{
				Description:   "Root directory in the repository for template deployments.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"status": schema.StringAttribute{
				Description: "The deployment build status.",
				Computed:    true,
			},
			"build_logs": schema.StringAttribute{
				Description: "The build logs.",
				Computed:    true,
			},
			"build_duration": schema.Int64Attribute{
				Description:   "The build duration in seconds.",
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"source_size": schema.Int64Attribute{
				Description:   "The source code size in bytes.",
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"build_size": schema.Int64Attribute{
				Description:   "The build output size in bytes.",
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"total_size": schema.Int64Attribute{
				Description:   "The total size in bytes.",
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Description:   "The deployment creation timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Description: "The deployment last update timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

func (r *deploymentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *deploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	sitesClient := appwrite.NewSites(r.clients.ClientForProject(projectID))

	siteID := plan.SiteID.ValueString()
	sourceType := plan.SourceType.ValueString()

	var deployment *models.Deployment

	switch sourceType {
	case "code":
		if plan.CodePath.IsNull() || plan.CodePath.IsUnknown() {
			resp.Diagnostics.AddError("Missing required attribute", `"code_path" is required when source_type is "code"`)
			return
		}
		codePath := plan.CodePath.ValueString()
		inputFile := appwritefile.NewInputFile(codePath, filepath.Base(codePath))

		var opts []sites.CreateDeploymentOption
		if !plan.InstallCommand.IsNull() && !plan.InstallCommand.IsUnknown() {
			opts = append(opts, sitesClient.WithCreateDeploymentInstallCommand(plan.InstallCommand.ValueString()))
		}
		if !plan.BuildCommand.IsNull() && !plan.BuildCommand.IsUnknown() {
			opts = append(opts, sitesClient.WithCreateDeploymentBuildCommand(plan.BuildCommand.ValueString()))
		}
		if !plan.OutputDirectory.IsNull() && !plan.OutputDirectory.IsUnknown() {
			opts = append(opts, sitesClient.WithCreateDeploymentOutputDirectory(plan.OutputDirectory.ValueString()))
		}
		if !plan.Activate.IsNull() && !plan.Activate.IsUnknown() {
			opts = append(opts, sitesClient.WithCreateDeploymentActivate(plan.Activate.ValueBool()))
		}

		deployment, err = sitesClient.CreateDeployment(siteID, inputFile, opts...)

	case "vcs":
		if plan.Type.IsNull() || plan.Type.IsUnknown() {
			resp.Diagnostics.AddError("Missing required attribute", `"type" is required when source_type is "vcs"`)
			return
		}
		if plan.Reference.IsNull() || plan.Reference.IsUnknown() {
			resp.Diagnostics.AddError("Missing required attribute", `"reference" is required when source_type is "vcs"`)
			return
		}

		var opts []sites.CreateVcsDeploymentOption
		if !plan.Activate.IsNull() && !plan.Activate.IsUnknown() {
			opts = append(opts, sitesClient.WithCreateVcsDeploymentActivate(plan.Activate.ValueBool()))
		}

		deployment, err = sitesClient.CreateVcsDeployment(siteID, plan.Type.ValueString(), plan.Reference.ValueString(), opts...)

	case "template":
		if plan.Repository.IsNull() || plan.Repository.IsUnknown() {
			resp.Diagnostics.AddError("Missing required attribute", `"repository" is required when source_type is "template"`)
			return
		}
		if plan.Owner.IsNull() || plan.Owner.IsUnknown() {
			resp.Diagnostics.AddError("Missing required attribute", `"owner" is required when source_type is "template"`)
			return
		}
		if plan.RootDirectory.IsNull() || plan.RootDirectory.IsUnknown() {
			resp.Diagnostics.AddError("Missing required attribute", `"root_directory" is required when source_type is "template"`)
			return
		}
		if plan.Type.IsNull() || plan.Type.IsUnknown() {
			resp.Diagnostics.AddError("Missing required attribute", `"type" is required when source_type is "template"`)
			return
		}
		if plan.Reference.IsNull() || plan.Reference.IsUnknown() {
			resp.Diagnostics.AddError("Missing required attribute", `"reference" is required when source_type is "template"`)
			return
		}

		var opts []sites.CreateTemplateDeploymentOption
		if !plan.Activate.IsNull() && !plan.Activate.IsUnknown() {
			opts = append(opts, sitesClient.WithCreateTemplateDeploymentActivate(plan.Activate.ValueBool()))
		}

		deployment, err = sitesClient.CreateTemplateDeployment(
			siteID,
			plan.Repository.ValueString(),
			plan.Owner.ValueString(),
			plan.RootDirectory.ValueString(),
			plan.Type.ValueString(),
			plan.Reference.ValueString(),
			opts...,
		)

	default:
		resp.Diagnostics.AddError("Invalid source_type", fmt.Sprintf("source_type must be one of: code, vcs, template. Got: %s", sourceType))
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error creating site deployment", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(deployment, &plan)

	// Wait for deployment to become ready if requested
	waitForReady := plan.WaitForReady.IsNull() || plan.WaitForReady.IsUnknown() || plan.WaitForReady.ValueBool()
	if waitForReady && deployment.Status != "ready" {
		err = common.WaitForDeploymentReady(ctx, func() (string, error) {
			d, err := sitesClient.GetDeployment(siteID, deployment.Id)
			if err != nil {
				return "", err
			}
			deployment = d
			r.mapToState(d, &plan)
			return d.Status, nil
		}, deployment.Id)
		if err != nil {
			resp.Diagnostics.AddError("Error waiting for site deployment", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *deploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	sitesClient := appwrite.NewSites(r.clients.ClientForProject(projectID))

	deployment, err := sitesClient.GetDeployment(state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading site deployment", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	r.mapToState(deployment, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *deploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Site deployments are immutable. All changes require replacement.")
}

func (r *deploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state deploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	sitesClient := appwrite.NewSites(r.clients.ClientForProject(projectID))

	_, err = sitesClient.DeleteDeployment(state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting site deployment", common.FormatError(err))
	}
}

func (r *deploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: site_id/deployment_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *deploymentResource) mapToState(deployment *models.Deployment, model *deploymentResourceModel) {
	model.ID = types.StringValue(deployment.Id)
	model.Status = types.StringValue(deployment.Status)
	model.BuildLogs = types.StringValue(deployment.BuildLogs)
	model.BuildDuration = types.Int64Value(int64(deployment.BuildDuration))
	model.SourceSize = types.Int64Value(int64(deployment.SourceSize))
	model.BuildSize = types.Int64Value(int64(deployment.BuildSize))
	model.TotalSize = types.Int64Value(int64(deployment.TotalSize))
	model.CreatedAt = types.StringValue(deployment.CreatedAt)
	model.UpdatedAt = types.StringValue(deployment.UpdatedAt)
}
