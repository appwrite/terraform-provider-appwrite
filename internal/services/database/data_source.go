package database

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v5/appwrite"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &databaseDataSource{}
	_ datasource.DataSourceWithConfigure = &databaseDataSource{}
)

type databaseDataSource struct {
	clients *common.AppwriteClients
}

type databaseDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Enabled   types.Bool   `tfsdk:"enabled"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	ProjectID types.String `tfsdk:"project_id"`
}

func NewDatabaseDataSource() datasource.DataSource {
	return &databaseDataSource{}
}

func (d *databaseDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tablesdb"
}

func (d *databaseDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an Appwrite database by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The database ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The database name.",
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the database is enabled.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The database creation timestamp.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The database last update timestamp.",
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

func (d *databaseDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *databaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config databaseDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(d.clients, config.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	tablesdbClient := appwrite.NewTablesDB(d.clients.ClientForProject(projectID))

	db, err := tablesdbClient.Get(config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading database", common.FormatError(err))
		return
	}

	config.ProjectID = types.StringValue(projectID)
	config.ID = types.StringValue(db.Id)
	config.Name = types.StringValue(db.Name)
	config.Enabled = types.BoolValue(db.Enabled)
	config.CreatedAt = types.StringValue(db.CreatedAt)
	config.UpdatedAt = types.StringValue(db.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
