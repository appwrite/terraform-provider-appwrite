package dedicateddatabase

import (
	"context"
	"fmt"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/aw-tests/sdk-for-go/v6/id"
	"github.com/aw-tests/sdk-for-go/v6/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &backupPolicyResource{}
	_ resource.ResourceWithConfigure   = &backupPolicyResource{}
	_ resource.ResourceWithImportState = &backupPolicyResource{}
)

type backupPolicyResource struct {
	clients *common.AppwriteClients
}

type backupPolicyResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Engine     types.String `tfsdk:"engine"`
	DatabaseID types.String `tfsdk:"database_id"`
	Name       types.String `tfsdk:"name"`
	Schedule   types.String `tfsdk:"schedule"`
	Retention  types.Int64  `tfsdk:"retention"`
	Type       types.String `tfsdk:"type"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
	ProjectID  types.String `tfsdk:"project_id"`
}

func NewBackupPolicyResource() resource.Resource {
	return &backupPolicyResource{}
}

func (r *backupPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_database_backup_policy"
}

func (r *backupPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	forceNewString := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages a backup policy for an Appwrite dedicated database.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The backup policy ID.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"engine": schema.StringAttribute{
				Description:   fmt.Sprintf("The engine of the target database. One of: %s.", validEngines()),
				Required:      true,
				Validators:    []validator.String{stringvalidator.OneOf("postgresql", "mysql", "mongo")},
				PlanModifiers: forceNewString,
			},
			"database_id": schema.StringAttribute{
				Description:   "The dedicated database this policy backs up.",
				Required:      true,
				PlanModifiers: forceNewString,
			},
			"name": schema.StringAttribute{
				Description: "The backup policy name.",
				Required:    true,
			},
			"schedule": schema.StringAttribute{
				Description: "Backup schedule in CRON format.",
				Required:    true,
			},
			"retention": schema.Int64Attribute{
				Description: "How many days to keep each backup before automatic deletion.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description:   "The backup type (e.g. full). Changing this forces a new policy.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: append([]planmodifier.String{stringplanmodifier.UseStateForUnknown()}, forceNewString...),
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the policy is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"created_at": schema.StringAttribute{
				Description: "The policy creation timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description:   "The policy last update timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{common.UseStateForUnknownUnlessUpdating()},
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

func (r *backupPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *backupPolicyResource) policyPath(engine, databaseID, policyID string) string {
	base := "/" + engine + "/" + databaseID + "/backups/policies"
	if policyID != "" {
		return base + "/" + policyID
	}
	return base
}

func (r *backupPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan backupPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	c := engineClient(r.clients, projectID)
	engine := plan.Engine.ValueString()

	policyID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		policyID = id.Unique()
	}

	params := map[string]interface{}{
		"policyId":  policyID,
		"name":      plan.Name.ValueString(),
		"schedule":  plan.Schedule.ValueString(),
		"retention": int(plan.Retention.ValueInt64()),
	}
	setStr(params, "type", plan.Type)
	setBool(params, "enabled", plan.Enabled)

	var policy models.BackupPolicy
	if err := apiCall(c, r.clients.UserAgent, "POST", r.policyPath(engine, plan.DatabaseID.ValueString(), ""), params, &policy); err != nil {
		resp.Diagnostics.AddError("Error creating backup policy", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(&policy, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *backupPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state backupPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	c := engineClient(r.clients, projectID)

	var policy models.BackupPolicy
	if err := apiCall(c, r.clients.UserAgent, "GET", r.policyPath(state.Engine.ValueString(), state.DatabaseID.ValueString(), state.ID.ValueString()), nil, &policy); err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading backup policy", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	r.mapToState(&policy, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *backupPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan backupPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	c := engineClient(r.clients, projectID)

	params := map[string]interface{}{
		"name":      plan.Name.ValueString(),
		"schedule":  plan.Schedule.ValueString(),
		"retention": int(plan.Retention.ValueInt64()),
	}
	setBool(params, "enabled", plan.Enabled)

	var policy models.BackupPolicy
	if err := apiCall(c, r.clients.UserAgent, "PATCH", r.policyPath(plan.Engine.ValueString(), plan.DatabaseID.ValueString(), plan.ID.ValueString()), params, &policy); err != nil {
		resp.Diagnostics.AddError("Error updating backup policy", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(&policy, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *backupPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state backupPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	c := engineClient(r.clients, projectID)

	if err := apiCall[any](c, r.clients.UserAgent, "DELETE", r.policyPath(state.Engine.ValueString(), state.DatabaseID.ValueString(), state.ID.ValueString()), nil, nil); err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting backup policy", common.FormatError(err))
	}
}

// ImportState expects "engine/database_id/policy_id".
func (r *backupPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	engine, rest, ok := splitTwo(req.ID)
	if ok {
		var dbID, policyID string
		dbID, policyID, ok = splitTwo(rest)
		if ok {
			if _, valid := engines[engine]; !valid {
				resp.Diagnostics.AddError("Invalid engine", fmt.Sprintf("Engine must be one of: %s", validEngines()))
				return
			}
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("engine"), engine)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_id"), dbID)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), policyID)...)
			return
		}
	}
	resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: engine/database_id/policy_id, got: %s", req.ID))
}

func (r *backupPolicyResource) mapToState(policy *models.BackupPolicy, model *backupPolicyResourceModel) {
	model.ID = types.StringValue(policy.Id)
	model.Name = types.StringValue(policy.Name)
	model.Schedule = types.StringValue(policy.Schedule)
	model.Retention = types.Int64Value(int64(policy.Retention))
	model.Type = types.StringValue(policy.Type)
	model.Enabled = types.BoolValue(policy.Enabled)
	model.CreatedAt = types.StringValue(policy.CreatedAt)
	model.UpdatedAt = types.StringValue(policy.UpdatedAt)
}
