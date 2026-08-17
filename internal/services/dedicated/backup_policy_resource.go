package dedicated

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/id"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

// backupPolicyResource manages a scheduled backup policy attached to one
// dedicated database. This is distinct from appwrite_backup_policy, which
// covers the shared-infrastructure Backups service.
type backupPolicyResource struct {
	engine  Engine
	clients *common.AppwriteClients
}

type backupPolicyResourceModel struct {
	ID         types.String `tfsdk:"id"`
	DatabaseID types.String `tfsdk:"database_id"`
	Name       types.String `tfsdk:"name"`
	Schedule   types.String `tfsdk:"schedule"`
	Retention  types.Int64  `tfsdk:"retention"`
	Type       types.String `tfsdk:"type"`
	Enabled    types.Bool   `tfsdk:"enabled"`

	Services     types.Set    `tfsdk:"services"`
	Resources    types.Set    `tfsdk:"resources"`
	ResourceID   types.String `tfsdk:"resource_id"`
	ResourceType types.String `tfsdk:"resource_type"`

	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	ProjectID types.String `tfsdk:"project_id"`
}

// NewBackupPolicyResource returns a constructor for the backup policy resource
// of one engine.
func NewBackupPolicyResource(engine Engine) func() resource.Resource {
	return func() resource.Resource {
		return &backupPolicyResource{engine: engine}
	}
}

func (r *backupPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_backup_policy", req.ProviderTypeName, r.engine)
}

func (r *backupPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf(
			"Manages a scheduled backup policy for a dedicated Appwrite %s database. Use `appwrite_backup_policy` instead for "+
				"databases running on Appwrite's shared infrastructure.",
			r.engine.Label(),
		),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The backup policy ID. Must be unique within the database. Generated when omitted.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"database_id": schema.StringAttribute{
				Description:   "The dedicated database ID the policy backs up.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description: "The backup policy name.",
				Required:    true,
			},
			"schedule": schema.StringAttribute{
				Description: "The backup schedule in CRON format, for example `0 3 * * *` for daily at 03:00 UTC.",
				Required:    true,
			},
			"retention": schema.Int64Attribute{
				Description: "How many days to keep each backup before it is deleted automatically.",
				Required:    true,
				Validators:  []validator.Int64{int64validator.AtLeast(1)},
			},
			"type": schema.StringAttribute{
				Description:   "The backup type. `full` takes a complete snapshot; `incremental` stores changes since the last backup. Changing this replaces the policy.",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.String{stringvalidator.OneOf("full", "incremental")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the policy is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},

			"services": schema.SetAttribute{
				Description: "The services covered by this policy.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"resources": schema.SetAttribute{
				Description: "The resources covered by this policy.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"resource_id":   schema.StringAttribute{Description: "The single resource the policy backs up, when scoped to one.", Computed: true},
			"resource_type": schema.StringAttribute{Description: "The type of the single resource the policy backs up.", Computed: true},

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
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData),
		)
		return
	}
	r.clients = clients
}

func (r *backupPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan backupPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	api := clientFor(r.clients, r.engine, projectID)

	policyID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		policyID = id.Unique()
	}

	policy, err := api.CreateBackupPolicy(
		plan.DatabaseID.ValueString(),
		policyID,
		plan.Name.ValueString(),
		plan.Schedule.ValueString(),
		int(plan.Retention.ValueInt64()),
		CreateBackupPolicyOptions{
			Type:    optString(plan.Type),
			Enabled: optBool(plan.Enabled),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating backup policy", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, policy, &plan, &resp.Diagnostics)
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
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	api := clientFor(r.clients, r.engine, projectID)

	policy, err := api.GetBackupPolicy(state.DatabaseID.ValueString(), state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading backup policy", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, policy, &state, &resp.Diagnostics)
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
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	api := clientFor(r.clients, r.engine, projectID)

	retention := int(plan.Retention.ValueInt64())
	policy, err := api.UpdateBackupPolicy(plan.DatabaseID.ValueString(), plan.ID.ValueString(), UpdateBackupPolicyOptions{
		Name:      optString(plan.Name),
		Schedule:  optString(plan.Schedule),
		Retention: &retention,
		Enabled:   optBool(plan.Enabled),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating backup policy", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, policy, &plan, &resp.Diagnostics)
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
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	api := clientFor(r.clients, r.engine, projectID)

	if err := api.DeleteBackupPolicy(state.DatabaseID.ValueString(), state.ID.ValueString()); err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting backup policy", common.FormatError(err))
	}
}

func (r *backupPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	databaseID, policyID, ok := splitImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: database_id/policy_id, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_id"), databaseID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), policyID)...)
}

func (r *backupPolicyResource) mapToState(ctx context.Context, policy *models.BackupPolicy, model *backupPolicyResourceModel, diagnostics *diag.Diagnostics) {
	model.ID = types.StringValue(policy.Id)
	model.Name = types.StringValue(policy.Name)
	model.Schedule = types.StringValue(policy.Schedule)
	model.Retention = types.Int64Value(int64(policy.Retention))
	model.Type = types.StringValue(policy.Type)
	model.Enabled = types.BoolValue(policy.Enabled)
	model.ResourceID = types.StringValue(policy.ResourceId)
	model.ResourceType = types.StringValue(policy.ResourceType)
	model.CreatedAt = types.StringValue(policy.CreatedAt)
	model.UpdatedAt = types.StringValue(policy.UpdatedAt)

	services, diags := types.SetValueFrom(ctx, types.StringType, nonNilStrings(policy.Services))
	diagnostics.Append(diags...)
	model.Services = services

	resources, diags := types.SetValueFrom(ctx, types.StringType, nonNilStrings(policy.Resources))
	diagnostics.Append(diags...)
	model.Resources = resources
}
