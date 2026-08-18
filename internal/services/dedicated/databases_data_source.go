package dedicated

import (
	"context"
	"fmt"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &databasesDataSource{}
	_ datasource.DataSourceWithConfigure = &databasesDataSource{}
)

type databasesDataSource struct {
	engine  Engine
	clients *common.AppwriteClients
}

type databasesDataSourceModel struct {
	ProjectID types.String      `tfsdk:"project_id"`
	Queries   types.List        `tfsdk:"queries"`
	Total     types.Int64       `tfsdk:"total"`
	Databases []databaseSummary `tfsdk:"databases"`
}

// databaseSummary carries the fields worth listing. The full record, including
// connection credentials, comes from the singular data source; repeating it for
// every database would put every password in state for a plain listing.
type databaseSummary struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Engine        types.String `tfsdk:"engine"`
	Version       types.String `tfsdk:"version"`
	Specification types.String `tfsdk:"specification"`
	Status        types.String `tfsdk:"status"`
	Hostname      types.String `tfsdk:"hostname"`
	Replicas      types.Int64  `tfsdk:"replicas"`
	CPU           types.Int64  `tfsdk:"cpu"`
	Memory        types.Int64  `tfsdk:"memory"`
	Storage       types.Int64  `tfsdk:"storage"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

// NewDatabasesDataSource returns a constructor for the plural databases data
// source of one engine.
func NewDatabasesDataSource(engine Engine) func() datasource.DataSource {
	return func() datasource.DataSource {
		return &databasesDataSource{engine: engine}
	}
}

func (d *databasesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_databases", req.ProviderTypeName, d.engine)
}

func (d *databasesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf(
			"Lists the dedicated Appwrite %s databases in a project. Connection credentials are deliberately not included; read "+
				"them from the singular `appwrite_%s_database` data source for the one database that needs them, so a listing does "+
				"not put every password into state.",
			d.engine.Label(), d.engine,
		),
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Appwrite project ID. Defaults to the provider-level project_id.",
				Optional:    true,
				Computed:    true,
			},
			"queries": schema.ListAttribute{
				Description: "Appwrite query strings used to filter the listing, for example `equal(\"status\", \"ready\")`.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"total": schema.Int64Attribute{
				Description: "The total number of databases matching the query.",
				Computed:    true,
			},
			"databases": schema.ListNestedAttribute{
				Description: "The databases that matched.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":            schema.StringAttribute{Description: "The database ID.", Computed: true},
						"name":          schema.StringAttribute{Description: "The database display name.", Computed: true},
						"engine":        schema.StringAttribute{Description: "The database engine.", Computed: true},
						"version":       schema.StringAttribute{Description: "The engine version.", Computed: true},
						"specification": schema.StringAttribute{Description: "The compute specification slug.", Computed: true},
						"status":        schema.StringAttribute{Description: "The database status.", Computed: true},
						"hostname":      schema.StringAttribute{Description: "The hostname to connect to.", Computed: true},
						"replicas":      schema.Int64Attribute{Description: "The number of high availability replicas.", Computed: true},
						"cpu":           schema.Int64Attribute{Description: "The allocated CPU in millicores.", Computed: true},
						"memory":        schema.Int64Attribute{Description: "The allocated memory in MB.", Computed: true},
						"storage":       schema.Int64Attribute{Description: "The allocated storage in GB.", Computed: true},
						"created_at":    schema.StringAttribute{Description: "The creation timestamp in ISO 8601 format.", Computed: true},
						"updated_at":    schema.StringAttribute{Description: "The last update timestamp in ISO 8601 format.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *databasesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *databasesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config databasesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(d.clients, config.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}

	var queries []string
	if !config.Queries.IsNull() && !config.Queries.IsUnknown() {
		resp.Diagnostics.Append(config.Queries.ElementsAs(ctx, &queries, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	list, err := clientFor(d.clients, d.engine, projectID).List(queries)
	if err != nil {
		resp.Diagnostics.AddError("Error listing dedicated databases", common.FormatError(err))
		return
	}

	config.ProjectID = types.StringValue(projectID)
	config.Total = types.Int64Value(int64(list.Total))
	config.Databases = make([]databaseSummary, 0, len(list.Databases))
	for _, db := range list.Databases {
		config.Databases = append(config.Databases, databaseSummary{
			ID:            types.StringValue(db.Id),
			Name:          types.StringValue(db.Name),
			Engine:        types.StringValue(db.Engine),
			Version:       types.StringValue(db.Version),
			Specification: types.StringValue(db.Specification),
			Status:        types.StringValue(db.Status),
			Hostname:      types.StringValue(db.Hostname),
			Replicas:      types.Int64Value(int64(db.Replicas)),
			CPU:           types.Int64Value(int64(db.Cpu)),
			Memory:        types.Int64Value(int64(db.Memory)),
			Storage:       types.Int64Value(int64(db.Storage)),
			CreatedAt:     types.StringValue(db.CreatedAt),
			UpdatedAt:     types.StringValue(db.UpdatedAt),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
