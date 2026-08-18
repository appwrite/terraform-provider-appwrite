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
	_ datasource.DataSource              = &statusDataSource{}
	_ datasource.DataSourceWithConfigure = &statusDataSource{}
)

// statusDataSource reports the live operational state of a dedicated database:
// health, replication, connections and storage. It reads a measurement taken at
// refresh time, so the values are a snapshot rather than configuration.
type statusDataSource struct {
	engine  Engine
	clients *common.AppwriteClients
}

type statusDataSourceModel struct {
	DatabaseID types.String `tfsdk:"database_id"`
	ProjectID  types.String `tfsdk:"project_id"`

	Health  types.String `tfsdk:"health"`
	Ready   types.Bool   `tfsdk:"ready"`
	Engine  types.String `tfsdk:"engine"`
	Version types.String `tfsdk:"version"`
	Uptime  types.Int64  `tfsdk:"uptime"`

	ConnectionsCurrent types.Int64 `tfsdk:"connections_current"`
	ConnectionsMax     types.Int64 `tfsdk:"connections_max"`

	SyncMode             types.String `tfsdk:"sync_mode"`
	EffectiveSyncMode    types.String `tfsdk:"effective_sync_mode"`
	SyncDegraded         types.Bool   `tfsdk:"sync_degraded"`
	SyncAcknowledgements types.Int64  `tfsdk:"sync_acknowledgements"`
	SyncStandbyCount     types.Int64  `tfsdk:"sync_standby_count"`
	SyncStateConfirmed   types.Bool   `tfsdk:"sync_state_confirmed"`

	Replicas []statusReplica `tfsdk:"replicas"`
	Volumes  []statusVolume  `tfsdk:"volumes"`
}

type statusReplica struct {
	Index       types.Int64   `tfsdk:"index"`
	Role        types.String  `tfsdk:"role"`
	Healthy     types.Bool    `tfsdk:"healthy"`
	Replicating types.Bool    `tfsdk:"replicating"`
	LagSeconds  types.Float64 `tfsdk:"lag_seconds"`
}

type statusVolume struct {
	Path        types.String `tfsdk:"path"`
	UsedPercent types.String `tfsdk:"used_percent"`
	Available   types.String `tfsdk:"available"`
	Mounted     types.Bool   `tfsdk:"mounted"`
}

// NewStatusDataSource returns a constructor for the status data source of one
// engine.
func NewStatusDataSource(engine Engine) func() datasource.DataSource {
	return func() datasource.DataSource {
		return &statusDataSource{engine: engine}
	}
}

func (d *statusDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_database_status", req.ProviderTypeName, d.engine)
}

func (d *statusDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf(
			"Reads the live operational state of a dedicated Appwrite %s database: health, replication, connection counts and "+
				"storage volumes.\n\n"+
				"These are measurements taken when Terraform refreshes, not configuration, so they change between runs on their "+
				"own. Use them for outputs and checks, not to drive resource arguments.",
			d.engine.Label(),
		),
		Attributes: map[string]schema.Attribute{
			"database_id": schema.StringAttribute{
				Description: "The dedicated database ID.",
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The Appwrite project ID. Defaults to the provider-level project_id.",
				Optional:    true,
				Computed:    true,
			},

			"health":  schema.StringAttribute{Description: "Overall health: `healthy`, `degraded`, `unhealthy`, or `unknown` when nothing could be measured.", Computed: true},
			"ready":   schema.BoolAttribute{Description: "Whether the database is ready to accept connections.", Computed: true},
			"engine":  schema.StringAttribute{Description: "The database engine.", Computed: true},
			"version": schema.StringAttribute{Description: "The engine version.", Computed: true},
			"uptime":  schema.Int64Attribute{Description: "Uptime in seconds.", Computed: true},

			"connections_current": schema.Int64Attribute{Description: "The current number of active connections.", Computed: true},
			"connections_max":     schema.Int64Attribute{Description: "The engine's own max_connections. On a pooled database this is the backend limit the pooler multiplexes onto, not the ceiling a client pool may reach.", Computed: true},

			"sync_mode":             schema.StringAttribute{Description: "The requested replication sync mode.", Computed: true},
			"effective_sync_mode":   schema.StringAttribute{Description: "The sync mode the primary is actually enforcing. Empty when high availability is disabled or the state could not be read.", Computed: true},
			"sync_degraded":         schema.BoolAttribute{Description: "Whether the enforced replication is weaker than the requested `sync_mode`.", Computed: true},
			"sync_acknowledgements": schema.Int64Attribute{Description: "How many standby acknowledgements the primary waits for before committing a write.", Computed: true},
			"sync_standby_count":    schema.Int64Attribute{Description: "How many standbys are registered with the primary for synchronous replication.", Computed: true},
			"sync_state_confirmed":  schema.BoolAttribute{Description: "Whether the sync fields are an engine reading rather than a recorded estimate. False means no reading was taken, not that replication is unhealthy — draw no conclusion about replication health from it.", Computed: true},

			"replicas": schema.ListNestedAttribute{
				Description: "Every configured member, including one the backend has not brought up, which is reported as not healthy.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"index":       schema.Int64Attribute{Description: "The member index. Read `role` for which member accepts writes: a failover moves the primary without renumbering.", Computed: true},
						"role":        schema.StringAttribute{Description: "`primary`, `replica`, or `unknown` while a transition is moving the topology.", Computed: true},
						"healthy":     schema.BoolAttribute{Description: "Whether the member is healthy.", Computed: true},
						"replicating": schema.BoolAttribute{Description: "Whether the member is streaming from the primary. False for a primary, which has no stream to report, and for an unhealthy member, which is not probed.", Computed: true},
						"lag_seconds": schema.Float64Attribute{Description: "Replication lag in seconds. 0 for the primary, and for a streaming member whose engine printed no numeric lag.", Computed: true},
					},
				},
			},
			"volumes": schema.ListNestedAttribute{
				Description: "The storage volumes backing the database.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"path":         schema.StringAttribute{Description: "The mount path.", Computed: true},
						"used_percent": schema.StringAttribute{Description: "The percentage of storage used.", Computed: true},
						"available":    schema.StringAttribute{Description: "The available storage space.", Computed: true},
						"mounted":      schema.BoolAttribute{Description: "Whether the volume is mounted.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *statusDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *statusDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config statusDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(d.clients, config.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}

	status, err := clientFor(d.clients, d.engine, projectID).GetStatus(config.DatabaseID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading dedicated database status", common.FormatError(err))
		return
	}

	config.ProjectID = types.StringValue(projectID)
	config.Health = types.StringValue(status.Health)
	config.Ready = types.BoolValue(status.Ready)
	config.Engine = types.StringValue(status.Engine)
	config.Version = types.StringValue(status.Version)
	config.Uptime = types.Int64Value(int64(status.Uptime))

	config.ConnectionsCurrent = types.Int64Value(int64(status.Connections.Current))
	config.ConnectionsMax = types.Int64Value(int64(status.Connections.Max))

	config.SyncMode = types.StringValue(status.SyncMode)
	config.EffectiveSyncMode = types.StringValue(status.EffectiveSyncMode)
	config.SyncDegraded = types.BoolValue(status.SyncDegraded)
	config.SyncAcknowledgements = types.Int64Value(int64(status.SyncAcknowledgements))
	config.SyncStandbyCount = types.Int64Value(int64(status.SyncStandbyCount))
	config.SyncStateConfirmed = types.BoolValue(status.SyncStateConfirmed)

	config.Replicas = make([]statusReplica, 0, len(status.Replicas))
	for _, replica := range status.Replicas {
		config.Replicas = append(config.Replicas, statusReplica{
			Index:       types.Int64Value(int64(replica.Index)),
			Role:        types.StringValue(replica.Role),
			Healthy:     types.BoolValue(replica.Healthy),
			Replicating: types.BoolValue(replica.Replicating),
			LagSeconds:  types.Float64Value(replica.LagSeconds),
		})
	}

	config.Volumes = make([]statusVolume, 0, len(status.Volumes))
	for _, volume := range status.Volumes {
		config.Volumes = append(config.Volumes, statusVolume{
			Path:        types.StringValue(volume.Path),
			UsedPercent: types.StringValue(volume.UsedPercent),
			Available:   types.StringValue(volume.Available),
			Mounted:     types.BoolValue(volume.Mounted),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
