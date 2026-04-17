package function

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v3/appwrite"
	"github.com/appwrite/sdk-for-go/v3/functions"
	"github.com/appwrite/sdk-for-go/v3/id"
	"github.com/appwrite/sdk-for-go/v3/models"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = &functionResource{}
	_ resource.ResourceWithConfigure   = &functionResource{}
	_ resource.ResourceWithImportState = &functionResource{}
)

type functionResource struct {
	clients *common.AppwriteClients
}

type functionResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Runtime              types.String `tfsdk:"runtime"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	Logging              types.Bool   `tfsdk:"logging"`
	Entrypoint           types.String `tfsdk:"entrypoint"`
	Commands             types.String `tfsdk:"commands"`
	Schedule             types.String `tfsdk:"schedule"`
	Timeout              types.Int64  `tfsdk:"timeout"`
	Execute              types.List   `tfsdk:"execute"`
	Events               types.List   `tfsdk:"events"`
	Scopes               types.List   `tfsdk:"scopes"`
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

func NewFunctionResource() resource.Resource {
	return &functionResource{}
}

func (r *functionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_function"
}

func (r *functionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite function.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The function ID.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The function name.",
				Required:    true,
			},
			"runtime": schema.StringAttribute{
				Description: "The function execution runtime (e.g. node-22, python-3.11, dart-3.5).",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the function is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"logging": schema.BoolAttribute{
				Description: "Whether execution logs are enabled. When disabled, executions will be slightly faster. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"entrypoint": schema.StringAttribute{
				Description: "The entrypoint file used to execute the deployment.",
				Optional:    true,
			},
			"commands": schema.StringAttribute{
				Description: "The build command used to build the deployment.",
				Optional:    true,
			},
			"schedule": schema.StringAttribute{
				Description: "Function execution schedule in CRON format.",
				Optional:    true,
			},
			"timeout": schema.Int64Attribute{
				Description: "Function execution timeout in seconds. Defaults to 15.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(15),
			},
			"execute": schema.ListAttribute{
				Description: "Execution permissions (e.g. users, teams, roles).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"events": schema.ListAttribute{
				Description: "Events that trigger the function.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"scopes": schema.ListAttribute{
				Description: "Allowed permission scopes.",
				Optional:    true,
				ElementType: types.StringType,
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
				Description: "Path to function in VCS repository.",
				Optional:    true,
			},
			"build_specification": schema.StringAttribute{
				Description:   "Machine specification for deployment builds.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"runtime_specification": schema.StringAttribute{
				Description:   "Machine specification for executions.",
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
				Description:   "The function creation timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Description: "The function last update timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

func (r *functionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *functionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan functionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	functionsClient := appwrite.NewFunctions(r.clients.ClientForProject(projectID))

	functionID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		functionID = id.Unique()
	}

	var opts []functions.CreateOption
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		opts = append(opts, functionsClient.WithCreateEnabled(plan.Enabled.ValueBool()))
	}
	if !plan.Logging.IsNull() && !plan.Logging.IsUnknown() {
		opts = append(opts, functionsClient.WithCreateLogging(plan.Logging.ValueBool()))
	}
	if !plan.Entrypoint.IsNull() {
		opts = append(opts, functionsClient.WithCreateEntrypoint(plan.Entrypoint.ValueString()))
	}
	if !plan.Commands.IsNull() {
		opts = append(opts, functionsClient.WithCreateCommands(plan.Commands.ValueString()))
	}
	if !plan.Schedule.IsNull() {
		opts = append(opts, functionsClient.WithCreateSchedule(plan.Schedule.ValueString()))
	}
	if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
		opts = append(opts, functionsClient.WithCreateTimeout(int(plan.Timeout.ValueInt64())))
	}
	if !plan.Execute.IsNull() {
		var execute []string
		resp.Diagnostics.Append(plan.Execute.ElementsAs(ctx, &execute, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, functionsClient.WithCreateExecute(execute))
	}
	if !plan.Events.IsNull() {
		var events []string
		resp.Diagnostics.Append(plan.Events.ElementsAs(ctx, &events, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, functionsClient.WithCreateEvents(events))
	}
	if !plan.Scopes.IsNull() {
		var scopes []string
		resp.Diagnostics.Append(plan.Scopes.ElementsAs(ctx, &scopes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, functionsClient.WithCreateScopes(scopes))
	}
	if !plan.InstallationID.IsNull() {
		opts = append(opts, functionsClient.WithCreateInstallationId(plan.InstallationID.ValueString()))
	}
	if !plan.ProviderRepositoryID.IsNull() {
		opts = append(opts, functionsClient.WithCreateProviderRepositoryId(plan.ProviderRepositoryID.ValueString()))
	}
	if !plan.ProviderBranch.IsNull() {
		opts = append(opts, functionsClient.WithCreateProviderBranch(plan.ProviderBranch.ValueString()))
	}
	if !plan.ProviderSilentMode.IsNull() {
		opts = append(opts, functionsClient.WithCreateProviderSilentMode(plan.ProviderSilentMode.ValueBool()))
	}
	if !plan.ProviderRootDir.IsNull() {
		opts = append(opts, functionsClient.WithCreateProviderRootDirectory(plan.ProviderRootDir.ValueString()))
	}
	if !plan.BuildSpecification.IsNull() && !plan.BuildSpecification.IsUnknown() {
		opts = append(opts, functionsClient.WithCreateBuildSpecification(plan.BuildSpecification.ValueString()))
	}
	if !plan.RuntimeSpecification.IsNull() && !plan.RuntimeSpecification.IsUnknown() {
		opts = append(opts, functionsClient.WithCreateRuntimeSpecification(plan.RuntimeSpecification.ValueString()))
	}
	if !plan.DeploymentRetention.IsNull() {
		opts = append(opts, functionsClient.WithCreateDeploymentRetention(int(plan.DeploymentRetention.ValueInt64())))
	}

	function, err := functionsClient.Create(functionID, plan.Name.ValueString(), plan.Runtime.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error creating function", common.FormatError(err))
		return
	}

	planned := plan

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, function, &plan, &resp.Diagnostics)

	checks := []common.AttrCheck{
		common.CheckBoolNotIgnored(planned.Logging, function.Logging, "logging", "function"),
		common.CheckStringNotIgnored(planned.BuildSpecification, function.BuildSpecification, "build_specification", "function"),
		common.CheckStringNotIgnored(planned.RuntimeSpecification, function.RuntimeSpecification, "runtime_specification", "function"),
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

func (r *functionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state functionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	functionsClient := appwrite.NewFunctions(r.clients.ClientForProject(projectID))

	function, err := functionsClient.Get(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading function", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, function, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *functionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan functionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	functionsClient := appwrite.NewFunctions(r.clients.ClientForProject(projectID))

	var opts []functions.UpdateOption
	if !plan.Runtime.IsNull() {
		opts = append(opts, functionsClient.WithUpdateRuntime(plan.Runtime.ValueString()))
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		opts = append(opts, functionsClient.WithUpdateEnabled(plan.Enabled.ValueBool()))
	}
	if !plan.Logging.IsNull() && !plan.Logging.IsUnknown() {
		opts = append(opts, functionsClient.WithUpdateLogging(plan.Logging.ValueBool()))
	}
	if !plan.Entrypoint.IsNull() {
		opts = append(opts, functionsClient.WithUpdateEntrypoint(plan.Entrypoint.ValueString()))
	}
	if !plan.Commands.IsNull() {
		opts = append(opts, functionsClient.WithUpdateCommands(plan.Commands.ValueString()))
	}
	if !plan.Schedule.IsNull() {
		opts = append(opts, functionsClient.WithUpdateSchedule(plan.Schedule.ValueString()))
	}
	if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
		opts = append(opts, functionsClient.WithUpdateTimeout(int(plan.Timeout.ValueInt64())))
	}
	if !plan.Execute.IsNull() {
		var execute []string
		resp.Diagnostics.Append(plan.Execute.ElementsAs(ctx, &execute, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, functionsClient.WithUpdateExecute(execute))
	}
	if !plan.Events.IsNull() {
		var events []string
		resp.Diagnostics.Append(plan.Events.ElementsAs(ctx, &events, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, functionsClient.WithUpdateEvents(events))
	}
	if !plan.Scopes.IsNull() {
		var scopes []string
		resp.Diagnostics.Append(plan.Scopes.ElementsAs(ctx, &scopes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, functionsClient.WithUpdateScopes(scopes))
	}
	if !plan.InstallationID.IsNull() {
		opts = append(opts, functionsClient.WithUpdateInstallationId(plan.InstallationID.ValueString()))
	}
	if !plan.ProviderRepositoryID.IsNull() {
		opts = append(opts, functionsClient.WithUpdateProviderRepositoryId(plan.ProviderRepositoryID.ValueString()))
	}
	if !plan.ProviderBranch.IsNull() {
		opts = append(opts, functionsClient.WithUpdateProviderBranch(plan.ProviderBranch.ValueString()))
	}
	if !plan.ProviderSilentMode.IsNull() {
		opts = append(opts, functionsClient.WithUpdateProviderSilentMode(plan.ProviderSilentMode.ValueBool()))
	}
	if !plan.ProviderRootDir.IsNull() {
		opts = append(opts, functionsClient.WithUpdateProviderRootDirectory(plan.ProviderRootDir.ValueString()))
	}
	if !plan.BuildSpecification.IsNull() && !plan.BuildSpecification.IsUnknown() {
		opts = append(opts, functionsClient.WithUpdateBuildSpecification(plan.BuildSpecification.ValueString()))
	}
	if !plan.RuntimeSpecification.IsNull() && !plan.RuntimeSpecification.IsUnknown() {
		opts = append(opts, functionsClient.WithUpdateRuntimeSpecification(plan.RuntimeSpecification.ValueString()))
	}
	if !plan.DeploymentRetention.IsNull() {
		opts = append(opts, functionsClient.WithUpdateDeploymentRetention(int(plan.DeploymentRetention.ValueInt64())))
	}

	function, err := functionsClient.Update(plan.ID.ValueString(), plan.Name.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error updating function", common.FormatError(err))
		return
	}

	planned := plan

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, function, &plan, &resp.Diagnostics)

	checks := []common.AttrCheck{
		common.CheckBoolNotIgnored(planned.Logging, function.Logging, "logging", "function"),
		common.CheckStringNotIgnored(planned.BuildSpecification, function.BuildSpecification, "build_specification", "function"),
		common.CheckStringNotIgnored(planned.RuntimeSpecification, function.RuntimeSpecification, "runtime_specification", "function"),
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

func (r *functionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state functionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	functionsClient := appwrite.NewFunctions(r.clients.ClientForProject(projectID))

	_, err = functionsClient.Delete(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting function", common.FormatError(err))
	}
}

func (r *functionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *functionResource) mapToState(ctx context.Context, function *models.Function, model *functionResourceModel, diagnostics *diag.Diagnostics) {
	model.ID = types.StringValue(function.Id)
	model.Name = types.StringValue(function.Name)
	model.Runtime = types.StringValue(function.Runtime)
	model.Enabled = types.BoolValue(function.Enabled)
	model.Logging = types.BoolValue(function.Logging)
	model.Timeout = types.Int64Value(int64(function.Timeout))
	model.DeploymentID = types.StringValue(function.DeploymentId)
	model.CreatedAt = types.StringValue(function.CreatedAt)
	model.UpdatedAt = types.StringValue(function.UpdatedAt)

	if function.Entrypoint != "" {
		model.Entrypoint = types.StringValue(function.Entrypoint)
	}
	if function.Commands != "" {
		model.Commands = types.StringValue(function.Commands)
	}
	if function.Schedule != "" {
		model.Schedule = types.StringValue(function.Schedule)
	}
	if function.InstallationId != "" {
		model.InstallationID = types.StringValue(function.InstallationId)
	}
	if function.ProviderRepositoryId != "" {
		model.ProviderRepositoryID = types.StringValue(function.ProviderRepositoryId)
	}
	if function.ProviderBranch != "" {
		model.ProviderBranch = types.StringValue(function.ProviderBranch)
	}
	if function.ProviderRootDirectory != "" {
		model.ProviderRootDir = types.StringValue(function.ProviderRootDirectory)
	}
	if !model.ProviderSilentMode.IsNull() {
		model.ProviderSilentMode = types.BoolValue(function.ProviderSilentMode)
	}
	model.BuildSpecification = types.StringValue(function.BuildSpecification)
	model.RuntimeSpecification = types.StringValue(function.RuntimeSpecification)
	if function.DeploymentRetention != 0 {
		model.DeploymentRetention = types.Int64Value(int64(function.DeploymentRetention))
	}

	if !model.Execute.IsNull() {
		executeList, diags := types.ListValueFrom(ctx, types.StringType, function.Execute)
		diagnostics.Append(diags...)
		model.Execute = executeList
	}
	if !model.Events.IsNull() {
		eventsList, diags := types.ListValueFrom(ctx, types.StringType, function.Events)
		diagnostics.Append(diags...)
		model.Events = eventsList
	}
	if !model.Scopes.IsNull() {
		scopesList, diags := types.ListValueFrom(ctx, types.StringType, function.Scopes)
		diagnostics.Append(diags...)
		model.Scopes = scopesList
	}
}
