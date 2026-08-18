package docdb

import (
	"context"
	"fmt"

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
	product Product
	clients *common.AppwriteClients
}

type databaseDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	Type          types.String `tfsdk:"type"`
	Status        types.String `tfsdk:"status"`
	Engine        types.String `tfsdk:"engine"`
	Specification types.String `tfsdk:"specification"`
	Replicas      types.Int64  `tfsdk:"replicas"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
	ProjectID     types.String `tfsdk:"project_id"`
}

// NewDatabaseDataSource returns a constructor for the database data source of
// one product.
func NewDatabaseDataSource(product Product) func() datasource.DataSource {
	return func() datasource.DataSource {
		return &databaseDataSource{product: product}
	}
}

func (d *databaseDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s", req.ProviderTypeName, d.product)
}

func (d *databaseDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Fetches an Appwrite %s database by ID.", d.product.Label()),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The database ID.",
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The Appwrite project ID. Defaults to the provider-level project_id.",
				Optional:    true,
				Computed:    true,
			},

			"name":          schema.StringAttribute{Description: "The database name.", Computed: true},
			"enabled":       schema.BoolAttribute{Description: "Whether the database is enabled.", Computed: true},
			"type":          schema.StringAttribute{Description: "The database type reported by the server.", Computed: true},
			"status":        schema.StringAttribute{Description: "The dedicated backing's lifecycle status. Empty when there is no dedicated backing.", Computed: true},
			"engine":        schema.StringAttribute{Description: "The engine the dedicated backing runs on. Empty when there is no dedicated backing.", Computed: true},
			"specification": schema.StringAttribute{Description: "The compute specification of the dedicated backing. Empty when there is no dedicated backing.", Computed: true},
			"replicas":      schema.Int64Attribute{Description: "The number of high availability replicas on the dedicated backing.", Computed: true},
			"created_at":    schema.StringAttribute{Description: "The database creation timestamp in ISO 8601 format.", Computed: true},
			"updated_at":    schema.StringAttribute{Description: "The database last update timestamp in ISO 8601 format.", Computed: true},
		},
	}
}

func (d *databaseDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData))
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

	database, err := apiFor(d.clients, d.product, projectID).Get(config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading database", common.FormatError(err))
		return
	}

	config.ProjectID = types.StringValue(projectID)
	config.ID = types.StringValue(database.Id)
	config.Name = types.StringValue(database.Name)
	config.Enabled = types.BoolValue(database.Enabled)
	config.Type = types.StringValue(database.Type)
	config.Status = types.StringValue(database.Status)
	config.Engine = types.StringValue(database.Engine)
	config.Specification = types.StringValue(database.Specification)
	config.Replicas = types.Int64Value(int64(database.Replicas))
	config.CreatedAt = types.StringValue(database.CreatedAt)
	config.UpdatedAt = types.StringValue(database.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
