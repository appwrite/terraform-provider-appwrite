package webhook

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v2/id"
	"github.com/appwrite/sdk-for-go/v2/models"
	"github.com/appwrite/sdk-for-go/v2/webhooks"
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
	_ resource.Resource                = &webhookResource{}
	_ resource.ResourceWithConfigure   = &webhookResource{}
	_ resource.ResourceWithImportState = &webhookResource{}
)

type webhookResource struct {
	webhooks  *webhooks.Webhooks
	projectID string
}

type webhookResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	URL          types.String `tfsdk:"url"`
	Events       types.List   `tfsdk:"events"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	Security     types.Bool   `tfsdk:"security"`
	HttpUser     types.String `tfsdk:"http_user"`
	HttpPass     types.String `tfsdk:"http_pass"`
	SignatureKey types.String `tfsdk:"signature_key"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
	ProjectID    types.String `tfsdk:"project_id"`
}

func NewWebhookResource() resource.Resource {
	return &webhookResource{}
}

func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite webhook.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The webhook ID.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The webhook name.",
				Required:    true,
			},
			"url": schema.StringAttribute{
				Description: "The webhook URL endpoint.",
				Required:    true,
			},
			"events": schema.ListAttribute{
				Description: "The webhook trigger events.",
				Required:    true,
				ElementType: types.StringType,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the webhook is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"security": schema.BoolAttribute{
				Description: "Whether SSL/TLS certificate verification is enabled. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"http_user": schema.StringAttribute{
				Description: "HTTP basic authentication username.",
				Optional:    true,
			},
			"http_pass": schema.StringAttribute{
				Description: "HTTP basic authentication password.",
				Optional:    true,
				Sensitive:   true,
			},
			"signature_key": schema.StringAttribute{
				Description: "Signature key for validating incoming webhooks.",
				Computed:    true,
				Sensitive:   true,
			},
			"created_at": schema.StringAttribute{
				Description: "The webhook creation timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The webhook last update timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

func (r *webhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData))
		return
	}
	r.webhooks = clients.Webhooks
	r.projectID = clients.ProjectID
}

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhookID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		webhookID = id.Unique()
	}

	var events []string
	resp.Diagnostics.Append(plan.Events.ElementsAs(ctx, &events, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var opts []webhooks.CreateOption
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		opts = append(opts, r.webhooks.WithCreateEnabled(plan.Enabled.ValueBool()))
	}
	if !plan.Security.IsNull() && !plan.Security.IsUnknown() {
		opts = append(opts, r.webhooks.WithCreateSecurity(plan.Security.ValueBool()))
	}
	if !plan.HttpUser.IsNull() {
		opts = append(opts, r.webhooks.WithCreateHttpUser(plan.HttpUser.ValueString()))
	}
	if !plan.HttpPass.IsNull() {
		opts = append(opts, r.webhooks.WithCreateHttpPass(plan.HttpPass.ValueString()))
	}

	webhook, err := r.webhooks.Create(webhookID, plan.URL.ValueString(), plan.Name.ValueString(), events, opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error creating webhook", common.FormatError(err))
		return
	}

	r.mapToState(ctx, webhook, &plan, &resp.Diagnostics)
	if plan.ProjectID.IsNull() || plan.ProjectID.IsUnknown() {
		plan.ProjectID = types.StringValue(r.projectID)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhook, err := r.webhooks.Get(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading webhook", common.FormatError(err))
		return
	}

	r.mapToState(ctx, webhook, &state, &resp.Diagnostics)
	if state.ProjectID.IsNull() || state.ProjectID.IsUnknown() {
		state.ProjectID = types.StringValue(r.projectID)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var events []string
	resp.Diagnostics.Append(plan.Events.ElementsAs(ctx, &events, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var opts []webhooks.UpdateOption
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		opts = append(opts, r.webhooks.WithUpdateEnabled(plan.Enabled.ValueBool()))
	}
	if !plan.Security.IsNull() && !plan.Security.IsUnknown() {
		opts = append(opts, r.webhooks.WithUpdateSecurity(plan.Security.ValueBool()))
	}
	if !plan.HttpUser.IsNull() {
		opts = append(opts, r.webhooks.WithUpdateHttpUser(plan.HttpUser.ValueString()))
	}
	if !plan.HttpPass.IsNull() {
		opts = append(opts, r.webhooks.WithUpdateHttpPass(plan.HttpPass.ValueString()))
	}

	webhook, err := r.webhooks.Update(plan.ID.ValueString(), plan.Name.ValueString(), plan.URL.ValueString(), events, opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error updating webhook", common.FormatError(err))
		return
	}

	r.mapToState(ctx, webhook, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.webhooks.Delete(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting webhook", common.FormatError(err))
	}
}

func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *webhookResource) mapToState(ctx context.Context, webhook *models.Webhook, model *webhookResourceModel, diagnostics *diag.Diagnostics) {
	model.ID = types.StringValue(webhook.Id)
	model.Name = types.StringValue(webhook.Name)
	model.URL = types.StringValue(webhook.Url)
	model.Enabled = types.BoolValue(webhook.Enabled)
	model.Security = types.BoolValue(webhook.Security)
	model.SignatureKey = types.StringValue(webhook.SignatureKey)
	model.CreatedAt = types.StringValue(webhook.CreatedAt)
	model.UpdatedAt = types.StringValue(webhook.UpdatedAt)

	if webhook.HttpUser != "" {
		model.HttpUser = types.StringValue(webhook.HttpUser)
	}
	// Don't overwrite http_pass - API may not return it

	eventsList, diags := types.ListValueFrom(ctx, types.StringType, webhook.Events)
	diagnostics.Append(diags...)
	model.Events = eventsList
}
