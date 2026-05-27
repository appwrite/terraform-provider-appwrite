package webhook

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v4/appwrite"
	"github.com/appwrite/sdk-for-go/v4/id"
	"github.com/appwrite/sdk-for-go/v4/models"
	"github.com/appwrite/sdk-for-go/v4/webhooks"
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
	clients *common.AppwriteClients
}

type webhookResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	URL          types.String `tfsdk:"url"`
	Events       types.List   `tfsdk:"events"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	TLS          types.Bool   `tfsdk:"tls"`
	AuthUsername types.String `tfsdk:"auth_username"`
	AuthPassword types.String `tfsdk:"auth_password"`
	Secret       types.String `tfsdk:"secret"`
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
			"tls": schema.BoolAttribute{
				Description: "Whether SSL/TLS certificate verification is enabled. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"auth_username": schema.StringAttribute{
				Description: "HTTP basic authentication username.",
				Optional:    true,
			},
			"auth_password": schema.StringAttribute{
				Description: "HTTP basic authentication password.",
				Optional:    true,
				Sensitive:   true,
			},
			"secret": schema.StringAttribute{
				Description: "Secret key for validating incoming webhooks.",
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
				PlanModifiers: []planmodifier.String{
					common.UseStateForUnknownUnlessUpdating(),
				},
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
	r.clients = clients
}

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	webhooksClient := appwrite.NewWebhooks(r.clients.ClientForProject(projectID))

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
		opts = append(opts, webhooksClient.WithCreateEnabled(plan.Enabled.ValueBool()))
	}
	if !plan.TLS.IsNull() && !plan.TLS.IsUnknown() {
		opts = append(opts, webhooksClient.WithCreateTls(plan.TLS.ValueBool()))
	}
	if !plan.AuthUsername.IsNull() {
		opts = append(opts, webhooksClient.WithCreateAuthUsername(plan.AuthUsername.ValueString()))
	}
	if !plan.AuthPassword.IsNull() {
		opts = append(opts, webhooksClient.WithCreateAuthPassword(plan.AuthPassword.ValueString()))
	}

	webhook, err := webhooksClient.Create(webhookID, plan.URL.ValueString(), plan.Name.ValueString(), events, opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error creating webhook", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, webhook, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	webhooksClient := appwrite.NewWebhooks(r.clients.ClientForProject(projectID))

	webhook, err := webhooksClient.Get(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading webhook", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, webhook, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	webhooksClient := appwrite.NewWebhooks(r.clients.ClientForProject(projectID))

	var events []string
	resp.Diagnostics.Append(plan.Events.ElementsAs(ctx, &events, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var opts []webhooks.UpdateOption
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		opts = append(opts, webhooksClient.WithUpdateEnabled(plan.Enabled.ValueBool()))
	}
	if !plan.TLS.IsNull() && !plan.TLS.IsUnknown() {
		opts = append(opts, webhooksClient.WithUpdateTls(plan.TLS.ValueBool()))
	}
	if !plan.AuthUsername.IsNull() {
		opts = append(opts, webhooksClient.WithUpdateAuthUsername(plan.AuthUsername.ValueString()))
	}
	if !plan.AuthPassword.IsNull() {
		opts = append(opts, webhooksClient.WithUpdateAuthPassword(plan.AuthPassword.ValueString()))
	}

	webhook, err := webhooksClient.Update(plan.ID.ValueString(), plan.Name.ValueString(), plan.URL.ValueString(), events, opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error updating webhook", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, webhook, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	webhooksClient := appwrite.NewWebhooks(r.clients.ClientForProject(projectID))

	_, err = webhooksClient.Delete(state.ID.ValueString())
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
	model.TLS = types.BoolValue(webhook.Tls)
	model.Secret = types.StringValue(webhook.Secret)
	model.CreatedAt = types.StringValue(webhook.CreatedAt)
	model.UpdatedAt = types.StringValue(webhook.UpdatedAt)

	if webhook.AuthUsername != "" {
		model.AuthUsername = types.StringValue(webhook.AuthUsername)
	}
	// Don't overwrite auth_password - API may not return it

	eventsList, diags := types.ListValueFrom(ctx, types.StringType, webhook.Events)
	diagnostics.Append(diags...)
	model.Events = eventsList
}
