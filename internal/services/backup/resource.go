package backup

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v2/backups"
	"github.com/appwrite/sdk-for-go/v2/id"
	"github.com/appwrite/sdk-for-go/v2/models"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &policyResource{}
	_ resource.ResourceWithConfigure   = &policyResource{}
	_ resource.ResourceWithImportState = &policyResource{}
)

type policyResource struct {
	backups *backups.Backups
}

type policyResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Services   types.List   `tfsdk:"services"`
	ResourceID types.String `tfsdk:"resource_id"`
	Retention  types.Int64  `tfsdk:"retention"`
	Schedule   types.String `tfsdk:"schedule"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

func NewPolicyResource() resource.Resource {
	return &policyResource{}
}

func (r *policyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tablesdb_backup_policy"
}

func (r *policyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite database backup policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The backup policy ID.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description: "The backup policy name.",
				Optional:    true,
			},
			"services": schema.ListAttribute{
				Description: "The services to back up. For example: databases.",
				Required:    true,
				ElementType: types.StringType,
			},
			"resource_id": schema.StringAttribute{
				Description: "The resource ID to back up. Set to back up a single database instead of all databases.",
				Optional:    true,
			},
			"retention": schema.Int64Attribute{
				Description: "How many days to keep the backup before automatic deletion.",
				Required:    true,
			},
			"schedule": schema.StringAttribute{
				Description: "Backup schedule in CRON format.",
				Required:    true,
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
				Description: "The policy last update timestamp in ISO 8601 format.",
				Computed:    true,
			},
		},
	}
}

func (r *policyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData))
		return
	}
	r.backups = clients.Backups
}

func (r *policyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan policyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var services []string
	resp.Diagnostics.Append(plan.Services.ElementsAs(ctx, &services, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var opts []backups.CreatePolicyOption
	if !plan.Name.IsNull() {
		opts = append(opts, r.backups.WithCreatePolicyName(plan.Name.ValueString()))
	}
	if !plan.ResourceID.IsNull() {
		opts = append(opts, r.backups.WithCreatePolicyResourceId(plan.ResourceID.ValueString()))
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		opts = append(opts, r.backups.WithCreatePolicyEnabled(plan.Enabled.ValueBool()))
	}

	policyID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		policyID = id.Unique()
	}

	policy, err := r.backups.CreatePolicy(
		policyID,
		services,
		int(plan.Retention.ValueInt64()),
		plan.Schedule.ValueString(),
		opts...,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating backup policy", common.FormatError(err))
		return
	}

	r.mapToState(ctx, policy, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *policyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state policyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.backups.GetPolicy(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading backup policy", common.FormatError(err))
		return
	}

	r.mapToState(ctx, policy, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *policyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan policyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var opts []backups.UpdatePolicyOption
	if !plan.Name.IsNull() {
		opts = append(opts, r.backups.WithUpdatePolicyName(plan.Name.ValueString()))
	}
	if !plan.Retention.IsNull() {
		opts = append(opts, r.backups.WithUpdatePolicyRetention(int(plan.Retention.ValueInt64())))
	}
	if !plan.Schedule.IsNull() {
		opts = append(opts, r.backups.WithUpdatePolicySchedule(plan.Schedule.ValueString()))
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		opts = append(opts, r.backups.WithUpdatePolicyEnabled(plan.Enabled.ValueBool()))
	}

	policy, err := r.backups.UpdatePolicy(plan.ID.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error updating backup policy", common.FormatError(err))
		return
	}

	r.mapToState(ctx, policy, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *policyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state policyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.backups.DeletePolicy(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting backup policy", common.FormatError(err))
	}
}

func (r *policyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *policyResource) mapToState(ctx context.Context, policy *models.BackupPolicy, model *policyResourceModel, diagnostics *diag.Diagnostics) {
	model.ID = types.StringValue(policy.Id)
	model.Retention = types.Int64Value(int64(policy.Retention))
	model.Schedule = types.StringValue(policy.Schedule)
	model.Enabled = types.BoolValue(policy.Enabled)
	model.CreatedAt = types.StringValue(policy.CreatedAt)
	model.UpdatedAt = types.StringValue(policy.UpdatedAt)

	if policy.Name != "" {
		model.Name = types.StringValue(policy.Name)
	}
	if policy.ResourceId != "" {
		model.ResourceID = types.StringValue(policy.ResourceId)
	}

	servicesList, diags := types.ListValueFrom(ctx, types.StringType, policy.Services)
	diagnostics.Append(diags...)
	model.Services = servicesList
}
