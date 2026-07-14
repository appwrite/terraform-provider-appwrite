package topic

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v6/appwrite"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &topicDataSource{}
	_ datasource.DataSourceWithConfigure = &topicDataSource{}
)

type topicDataSource struct {
	clients *common.AppwriteClients
}

type topicDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Subscribe types.List   `tfsdk:"subscribe"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	ProjectID types.String `tfsdk:"project_id"`
}

func NewTopicDataSource() datasource.DataSource {
	return &topicDataSource{}
}

func (d *topicDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_messaging_topic"
}

func (d *topicDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an Appwrite messaging topic by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The topic ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The topic name.",
				Computed:    true,
			},
			"subscribe": schema.ListAttribute{
				Description: "Subscribe permissions.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.StringAttribute{
				Description: "The topic creation timestamp.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The topic last update timestamp.",
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

func (d *topicDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *topicDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config topicDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(d.clients, config.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	messagingClient := appwrite.NewMessaging(d.clients.ClientForProject(projectID))

	topic, err := messagingClient.GetTopic(config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading topic", common.FormatError(err))
		return
	}

	config.ProjectID = types.StringValue(projectID)
	config.ID = types.StringValue(topic.Id)
	config.Name = types.StringValue(topic.Name)
	config.CreatedAt = types.StringValue(topic.CreatedAt)
	config.UpdatedAt = types.StringValue(topic.UpdatedAt)

	subList, diags := types.ListValueFrom(ctx, types.StringType, topic.Subscribe)
	resp.Diagnostics.Append(diags...)
	config.Subscribe = subList

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
