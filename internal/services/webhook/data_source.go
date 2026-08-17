package webhook

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &webhookDataSource{}
	_ datasource.DataSourceWithConfigure = &webhookDataSource{}
)

type webhookDataSource struct {
	clients *common.AppwriteClients
}

type webhookDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	URL       types.String `tfsdk:"url"`
	Events    types.List   `tfsdk:"events"`
	Enabled   types.Bool   `tfsdk:"enabled"`
	TLS       types.Bool   `tfsdk:"tls"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	ProjectID types.String `tfsdk:"project_id"`
}

func NewWebhookDataSource() datasource.DataSource {
	return &webhookDataSource{}
}

func (d *webhookDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (d *webhookDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an Appwrite webhook by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The webhook ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The webhook name.",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "The webhook URL.",
				Computed:    true,
			},
			"events": schema.ListAttribute{
				Description: "Events that trigger the webhook.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the webhook is enabled.",
				Computed:    true,
			},
			"tls": schema.BoolAttribute{
				Description: "Whether TLS verification is enabled.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The webhook creation timestamp.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The webhook last update timestamp.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The Appwrite project ID. Defaults to the provider-level project_id.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (d *webhookDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData),
		)
		return
	}
	d.clients = clients
}

func (d *webhookDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config webhookDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(d.clients, config.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	webhooksClient := appwrite.NewWebhooks(d.clients.ClientForProject(projectID))

	webhook, err := webhooksClient.Get(config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook", common.FormatError(err))
		return
	}

	config.ProjectID = types.StringValue(projectID)
	config.ID = types.StringValue(webhook.Id)
	config.Name = types.StringValue(webhook.Name)
	config.URL = types.StringValue(webhook.Url)
	config.Enabled = types.BoolValue(webhook.Enabled)
	config.TLS = types.BoolValue(webhook.Tls)
	config.CreatedAt = types.StringValue(webhook.CreatedAt)
	config.UpdatedAt = types.StringValue(webhook.UpdatedAt)

	eventsList, diags := types.ListValueFrom(ctx, types.StringType, webhook.Events)
	resp.Diagnostics.Append(diags...)
	config.Events = eventsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
