package health

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v2/health"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &healthDataSource{}
	_ datasource.DataSourceWithConfigure = &healthDataSource{}
)

type healthDataSource struct {
	health *health.Health
}

type healthDataSourceModel struct {
	Name   types.String `tfsdk:"name"`
	Ping   types.Int64  `tfsdk:"ping"`
	Status types.String `tfsdk:"status"`
}

func NewHealthDataSource() datasource.DataSource {
	return &healthDataSource{}
}

func (d *healthDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_health"
}

func (d *healthDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns the Appwrite server health status.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the service.",
				Computed:    true,
			},
			"ping": schema.Int64Attribute{
				Description: "Duration in milliseconds how long the health check took.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Service status. Possible values are: pass, fail.",
				Computed:    true,
			},
		},
	}
}

func (d *healthDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData))
		return
	}
	d.health = clients.Health
}

func (d *healthDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	status, err := d.health.Get()
	if err != nil {
		resp.Diagnostics.AddError("Error reading health status", common.FormatError(err))
		return
	}

	state := healthDataSourceModel{
		Name:   types.StringValue(status.Name),
		Ping:   types.Int64Value(int64(status.Ping)),
		Status: types.StringValue(status.Status),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
