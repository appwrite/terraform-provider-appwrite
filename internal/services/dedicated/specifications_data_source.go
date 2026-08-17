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
	_ datasource.DataSource              = &specificationsDataSource{}
	_ datasource.DataSourceWithConfigure = &specificationsDataSource{}
)

type specificationsDataSource struct {
	engine  Engine
	clients *common.AppwriteClients
}

type specificationsDataSourceModel struct {
	ProjectID      types.String        `tfsdk:"project_id"`
	Specifications []specificationItem `tfsdk:"specifications"`
}

type specificationItem struct {
	Slug              types.String  `tfsdk:"slug"`
	Name              types.String  `tfsdk:"name"`
	Price             types.Float64 `tfsdk:"price"`
	CPU               types.Int64   `tfsdk:"cpu"`
	Memory            types.Int64   `tfsdk:"memory"`
	MaxConnections    types.Int64   `tfsdk:"max_connections"`
	IncludedStorage   types.Int64   `tfsdk:"included_storage"`
	IncludedBandwidth types.Int64   `tfsdk:"included_bandwidth"`
	Enabled           types.Bool    `tfsdk:"enabled"`
}

// NewSpecificationsDataSource returns a constructor for the specifications data
// source of one engine.
func NewSpecificationsDataSource(engine Engine) func() datasource.DataSource {
	return func() datasource.DataSource {
		return &specificationsDataSource{engine: engine}
	}
}

func (d *specificationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_specifications", req.ProviderTypeName, d.engine)
}

func (d *specificationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf(
			"Lists the compute specifications available for dedicated %s databases, so a `specification` slug can be selected "+
				"without hardcoding it. Availability depends on the organization's billing plan; check `enabled` before using a slug.",
			d.engine.Label(),
		),
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Appwrite project ID. Defaults to the provider-level project_id.",
				Optional:    true,
				Computed:    true,
			},
			"specifications": schema.ListNestedAttribute{
				Description: "The available specifications.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"slug":               schema.StringAttribute{Description: "The slug to pass as `specification` when creating a database.", Computed: true},
						"name":               schema.StringAttribute{Description: "The human readable specification name.", Computed: true},
						"price":              schema.Float64Attribute{Description: "The monthly price in USD.", Computed: true},
						"cpu":                schema.Int64Attribute{Description: "The allocated CPU in millicores.", Computed: true},
						"memory":             schema.Int64Attribute{Description: "The allocated memory in MB.", Computed: true},
						"max_connections":    schema.Int64Attribute{Description: "The maximum number of concurrent connections.", Computed: true},
						"included_storage":   schema.Int64Attribute{Description: "The included storage in GB before overage charges apply.", Computed: true},
						"included_bandwidth": schema.Int64Attribute{Description: "The included bandwidth in GB before overage charges apply.", Computed: true},
						"enabled":            schema.BoolAttribute{Description: "Whether the specification is available on the current billing plan.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *specificationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *specificationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config specificationsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(d.clients, config.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}

	list, err := clientFor(d.clients, d.engine, projectID).ListSpecifications()
	if err != nil {
		resp.Diagnostics.AddError("Error listing dedicated database specifications", common.FormatError(err))
		return
	}

	config.ProjectID = types.StringValue(projectID)
	config.Specifications = make([]specificationItem, 0, len(list.Specifications))
	for _, spec := range list.Specifications {
		config.Specifications = append(config.Specifications, specificationItem{
			Slug:              types.StringValue(spec.Slug),
			Name:              types.StringValue(spec.Name),
			Price:             types.Float64Value(spec.Price),
			CPU:               types.Int64Value(int64(spec.Cpu)),
			Memory:            types.Int64Value(int64(spec.Memory)),
			MaxConnections:    types.Int64Value(int64(spec.MaxConnections)),
			IncludedStorage:   types.Int64Value(int64(spec.IncludedStorage)),
			IncludedBandwidth: types.Int64Value(int64(spec.IncludedBandwidth)),
			Enabled:           types.BoolValue(spec.Enabled),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
