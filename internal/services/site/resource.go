package site

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v4/appwrite"
	"github.com/appwrite/sdk-for-go/v4/id"
	"github.com/appwrite/sdk-for-go/v4/models"
	"github.com/appwrite/sdk-for-go/v4/sites"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &siteResource{}
	_ resource.ResourceWithConfigure   = &siteResource{}
	_ resource.ResourceWithImportState = &siteResource{}
)

type siteResource struct {
	clients *common.AppwriteClients
}

type siteResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Framework            types.String `tfsdk:"framework"`
	BuildRuntime         types.String `tfsdk:"build_runtime"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	Logging              types.Bool   `tfsdk:"logging"`
	Timeout              types.Int64  `tfsdk:"timeout"`
	InstallCommand       types.String `tfsdk:"install_command"`
	BuildCommand         types.String `tfsdk:"build_command"`
	StartCommand         types.String `tfsdk:"start_command"`
	OutputDirectory      types.String `tfsdk:"output_directory"`
	Adapter              types.String `tfsdk:"adapter"`
	FallbackFile         types.String `tfsdk:"fallback_file"`
	InstallationID       types.String `tfsdk:"installation_id"`
	ProviderRepositoryID types.String `tfsdk:"provider_repository_id"`
	ProviderBranch       types.String `tfsdk:"provider_branch"`
	ProviderSilentMode   types.Bool   `tfsdk:"provider_silent_mode"`
	ProviderRootDir      types.String `tfsdk:"provider_root_directory"`
	BuildSpecification   types.String `tfsdk:"build_specification"`
	RuntimeSpecification types.String `tfsdk:"runtime_specification"`
	DeploymentRetention  types.Int64  `tfsdk:"deployment_retention"`
	DeploymentID         types.String `tfsdk:"deployment_id"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
	ProjectID            types.String `tfsdk:"project_id"`
}

func NewSiteResource() resource.Resource {
	return &siteResource{}
}

func (r *siteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site"
}

func (r *siteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite site.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The site ID.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The site name.",
				Required:    true,
			},
			"framework": schema.StringAttribute{
				Description: "The site framework (e.g. nextjs, nuxt, sveltekit, astro, remix, analog, flutter, react, vue, vite, other).",
				Required:    true,
			},
			"build_runtime": schema.StringAttribute{
				Description: "The build runtime (e.g. node-22).",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the site is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"logging": schema.BoolAttribute{
				Description: "Whether request logs are enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"timeout": schema.Int64Attribute{
				Description: "Site request timeout in seconds. Defaults to 15.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(15),
			},
			"install_command": schema.StringAttribute{
				Description: "The install command used to install site dependencies.",
				Optional:    true,
			},
			"build_command": schema.StringAttribute{
				Description: "The build command used to build the site.",
				Optional:    true,
			},
			"start_command": schema.StringAttribute{
				Description: "Custom command to use when starting the site runtime.",
				Optional:    true,
			},
			"output_directory": schema.StringAttribute{
				Description: "The directory where the site build output is located.",
				Optional:    true,
			},
			"adapter": schema.StringAttribute{
				Description:   "The site framework adapter.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"fallback_file": schema.StringAttribute{
				Description: "Name of fallback file to use instead of 404 page. If null, Appwrite 404 page will be displayed.",
				Optional:    true,
			},
			"installation_id": schema.StringAttribute{
				Description: "VCS (Version Control System) installation ID.",
				Optional:    true,
			},
			"provider_repository_id": schema.StringAttribute{
				Description: "VCS (Version Control System) repository ID.",
				Optional:    true,
			},
			"provider_branch": schema.StringAttribute{
				Description: "VCS (Version Control System) branch name.",
				Optional:    true,
			},
			"provider_silent_mode": schema.BoolAttribute{
				Description: "Whether VCS silent mode is enabled (no comments on pull requests).",
				Optional:    true,
			},
			"provider_root_directory": schema.StringAttribute{
				Description: "Path to site in VCS repository.",
				Optional:    true,
			},
			"build_specification": schema.StringAttribute{
				Description:   "Machine specification for deployment builds.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"runtime_specification": schema.StringAttribute{
				Description:   "Machine specification for SSR executions.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"deployment_retention": schema.Int64Attribute{
				Description: "How many days to keep non-active deployments before automatic deletion.",
				Optional:    true,
			},
			"deployment_id": schema.StringAttribute{
				Description:   "The active deployment ID.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Description:   "The site creation timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Description: "The site last update timestamp in ISO 8601 format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					common.UseStateForUnknownUnlessUpdating(),
				},
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

func (r *siteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *siteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan siteResourceModel
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

	siteID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		siteID = id.Unique()
	}

	var opts []sites.CreateOption
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		opts = append(opts, sitesClient.WithCreateEnabled(plan.Enabled.ValueBool()))
	}
	if !plan.Logging.IsNull() && !plan.Logging.IsUnknown() {
		opts = append(opts, sitesClient.WithCreateLogging(plan.Logging.ValueBool()))
	}
	if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
		opts = append(opts, sitesClient.WithCreateTimeout(int(plan.Timeout.ValueInt64())))
	}
	if !plan.InstallCommand.IsNull() {
		opts = append(opts, sitesClient.WithCreateInstallCommand(plan.InstallCommand.ValueString()))
	}
	if !plan.BuildCommand.IsNull() {
		opts = append(opts, sitesClient.WithCreateBuildCommand(plan.BuildCommand.ValueString()))
	}
	if !plan.StartCommand.IsNull() {
		opts = append(opts, sitesClient.WithCreateStartCommand(plan.StartCommand.ValueString()))
	}
	if !plan.OutputDirectory.IsNull() {
		opts = append(opts, sitesClient.WithCreateOutputDirectory(plan.OutputDirectory.ValueString()))
	}
	if !plan.Adapter.IsNull() && !plan.Adapter.IsUnknown() && plan.Adapter.ValueString() != "" {
		opts = append(opts, sitesClient.WithCreateAdapter(plan.Adapter.ValueString()))
	}
	if !plan.FallbackFile.IsNull() {
		opts = append(opts, sitesClient.WithCreateFallbackFile(plan.FallbackFile.ValueString()))
	}
	if !plan.InstallationID.IsNull() {
		opts = append(opts, sitesClient.WithCreateInstallationId(plan.InstallationID.ValueString()))
	}
	if !plan.ProviderRepositoryID.IsNull() {
		opts = append(opts, sitesClient.WithCreateProviderRepositoryId(plan.ProviderRepositoryID.ValueString()))
	}
	if !plan.ProviderBranch.IsNull() {
		opts = append(opts, sitesClient.WithCreateProviderBranch(plan.ProviderBranch.ValueString()))
	}
	if !plan.ProviderSilentMode.IsNull() {
		opts = append(opts, sitesClient.WithCreateProviderSilentMode(plan.ProviderSilentMode.ValueBool()))
	}
	if !plan.ProviderRootDir.IsNull() {
		opts = append(opts, sitesClient.WithCreateProviderRootDirectory(plan.ProviderRootDir.ValueString()))
	}
	if !plan.BuildSpecification.IsNull() && !plan.BuildSpecification.IsUnknown() {
		opts = append(opts, sitesClient.WithCreateBuildSpecification(plan.BuildSpecification.ValueString()))
	}
	if !plan.RuntimeSpecification.IsNull() && !plan.RuntimeSpecification.IsUnknown() {
		opts = append(opts, sitesClient.WithCreateRuntimeSpecification(plan.RuntimeSpecification.ValueString()))
	}
	if !plan.DeploymentRetention.IsNull() {
		opts = append(opts, sitesClient.WithCreateDeploymentRetention(int(plan.DeploymentRetention.ValueInt64())))
	}

	site, err := sitesClient.Create(siteID, plan.Name.ValueString(), plan.Framework.ValueString(), plan.BuildRuntime.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error creating site", common.FormatError(err))
		return
	}

	planned := plan

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(site, &plan)

	checks := []common.AttrCheck{
		common.CheckBoolNotIgnored(planned.Logging, site.Logging, "logging", "site"),
		common.CheckStringNotIgnored(planned.Adapter, site.Adapter, "adapter", "site"),
		common.CheckStringNotIgnored(planned.BuildSpecification, site.BuildSpecification, "build_specification", "site"),
		common.CheckStringNotIgnored(planned.RuntimeSpecification, site.RuntimeSpecification, "runtime_specification", "site"),
	}
	for _, c := range checks {
		if c.Mismatch {
			resp.Diagnostics.AddError(c.Summary, c.Detail)
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *siteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state siteResourceModel
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

	site, err := sitesClient.Get(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading site", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	r.mapToState(site, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *siteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan siteResourceModel
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

	var opts []sites.UpdateOption
	if !plan.BuildRuntime.IsNull() {
		opts = append(opts, sitesClient.WithUpdateBuildRuntime(plan.BuildRuntime.ValueString()))
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		opts = append(opts, sitesClient.WithUpdateEnabled(plan.Enabled.ValueBool()))
	}
	if !plan.Logging.IsNull() && !plan.Logging.IsUnknown() {
		opts = append(opts, sitesClient.WithUpdateLogging(plan.Logging.ValueBool()))
	}
	if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
		opts = append(opts, sitesClient.WithUpdateTimeout(int(plan.Timeout.ValueInt64())))
	}
	if !plan.InstallCommand.IsNull() {
		opts = append(opts, sitesClient.WithUpdateInstallCommand(plan.InstallCommand.ValueString()))
	}
	if !plan.BuildCommand.IsNull() {
		opts = append(opts, sitesClient.WithUpdateBuildCommand(plan.BuildCommand.ValueString()))
	}
	if !plan.StartCommand.IsNull() {
		opts = append(opts, sitesClient.WithUpdateStartCommand(plan.StartCommand.ValueString()))
	}
	if !plan.OutputDirectory.IsNull() {
		opts = append(opts, sitesClient.WithUpdateOutputDirectory(plan.OutputDirectory.ValueString()))
	}
	if !plan.Adapter.IsNull() && !plan.Adapter.IsUnknown() && plan.Adapter.ValueString() != "" {
		opts = append(opts, sitesClient.WithUpdateAdapter(plan.Adapter.ValueString()))
	}
	if !plan.FallbackFile.IsNull() {
		opts = append(opts, sitesClient.WithUpdateFallbackFile(plan.FallbackFile.ValueString()))
	}
	if !plan.InstallationID.IsNull() {
		opts = append(opts, sitesClient.WithUpdateInstallationId(plan.InstallationID.ValueString()))
	}
	if !plan.ProviderRepositoryID.IsNull() {
		opts = append(opts, sitesClient.WithUpdateProviderRepositoryId(plan.ProviderRepositoryID.ValueString()))
	}
	if !plan.ProviderBranch.IsNull() {
		opts = append(opts, sitesClient.WithUpdateProviderBranch(plan.ProviderBranch.ValueString()))
	}
	if !plan.ProviderSilentMode.IsNull() {
		opts = append(opts, sitesClient.WithUpdateProviderSilentMode(plan.ProviderSilentMode.ValueBool()))
	}
	if !plan.ProviderRootDir.IsNull() {
		opts = append(opts, sitesClient.WithUpdateProviderRootDirectory(plan.ProviderRootDir.ValueString()))
	}
	if !plan.BuildSpecification.IsNull() && !plan.BuildSpecification.IsUnknown() {
		opts = append(opts, sitesClient.WithUpdateBuildSpecification(plan.BuildSpecification.ValueString()))
	}
	if !plan.RuntimeSpecification.IsNull() && !plan.RuntimeSpecification.IsUnknown() {
		opts = append(opts, sitesClient.WithUpdateRuntimeSpecification(plan.RuntimeSpecification.ValueString()))
	}
	if !plan.DeploymentRetention.IsNull() {
		opts = append(opts, sitesClient.WithUpdateDeploymentRetention(int(plan.DeploymentRetention.ValueInt64())))
	}

	site, err := sitesClient.Update(plan.ID.ValueString(), plan.Name.ValueString(), plan.Framework.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error updating site", common.FormatError(err))
		return
	}

	planned := plan

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(site, &plan)

	checks := []common.AttrCheck{
		common.CheckBoolNotIgnored(planned.Logging, site.Logging, "logging", "site"),
		common.CheckStringNotIgnored(planned.Adapter, site.Adapter, "adapter", "site"),
		common.CheckStringNotIgnored(planned.BuildSpecification, site.BuildSpecification, "build_specification", "site"),
		common.CheckStringNotIgnored(planned.RuntimeSpecification, site.RuntimeSpecification, "runtime_specification", "site"),
	}
	for _, c := range checks {
		if c.Mismatch {
			resp.Diagnostics.AddError(c.Summary, c.Detail)
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *siteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state siteResourceModel
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

	_, err = sitesClient.Delete(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting site", common.FormatError(err))
	}
}

func (r *siteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *siteResource) mapToState(site *models.Site, model *siteResourceModel) {
	model.ID = types.StringValue(site.Id)
	model.Name = types.StringValue(site.Name)
	model.Framework = types.StringValue(site.Framework)
	model.BuildRuntime = types.StringValue(site.BuildRuntime)
	model.Enabled = types.BoolValue(site.Enabled)
	model.Logging = types.BoolValue(site.Logging)
	model.Timeout = types.Int64Value(int64(site.Timeout))
	model.DeploymentID = types.StringValue(site.DeploymentId)
	model.BuildSpecification = types.StringValue(site.BuildSpecification)
	model.RuntimeSpecification = types.StringValue(site.RuntimeSpecification)
	model.CreatedAt = types.StringValue(site.CreatedAt)
	model.UpdatedAt = types.StringValue(site.UpdatedAt)

	model.Adapter = types.StringValue(site.Adapter)
	if site.InstallCommand != "" {
		model.InstallCommand = types.StringValue(site.InstallCommand)
	}
	if site.BuildCommand != "" {
		model.BuildCommand = types.StringValue(site.BuildCommand)
	}
	if site.StartCommand != "" {
		model.StartCommand = types.StringValue(site.StartCommand)
	}
	if site.OutputDirectory != "" {
		model.OutputDirectory = types.StringValue(site.OutputDirectory)
	}
	if site.FallbackFile != "" {
		model.FallbackFile = types.StringValue(site.FallbackFile)
	}
	if site.InstallationId != "" {
		model.InstallationID = types.StringValue(site.InstallationId)
	}
	if site.ProviderRepositoryId != "" {
		model.ProviderRepositoryID = types.StringValue(site.ProviderRepositoryId)
	}
	if site.ProviderBranch != "" {
		model.ProviderBranch = types.StringValue(site.ProviderBranch)
	}
	if site.ProviderRootDirectory != "" {
		model.ProviderRootDir = types.StringValue(site.ProviderRootDirectory)
	}
	if !model.ProviderSilentMode.IsNull() {
		model.ProviderSilentMode = types.BoolValue(site.ProviderSilentMode)
	}
	if site.DeploymentRetention != 0 {
		model.DeploymentRetention = types.Int64Value(int64(site.DeploymentRetention))
	}
}
