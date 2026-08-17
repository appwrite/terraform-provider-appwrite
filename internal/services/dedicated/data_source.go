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
	_ datasource.DataSource              = &databaseDataSource{}
	_ datasource.DataSourceWithConfigure = &databaseDataSource{}
)

type databaseDataSource struct {
	engine  Engine
	clients *common.AppwriteClients
}

type databaseDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Version       types.String `tfsdk:"version"`
	Specification types.String `tfsdk:"specification"`
	Status        types.String `tfsdk:"status"`

	Replicas types.Int64  `tfsdk:"replicas"`
	SyncMode types.String `tfsdk:"sync_mode"`

	NetworkIdleTimeoutSeconds types.Int64 `tfsdk:"network_idle_timeout_seconds"`
	NetworkIPAllowlist        types.Set   `tfsdk:"network_ip_allowlist"`
	NetworkMaxConnections     types.Int64 `tfsdk:"network_max_connections"`
	IdleTimeoutMinutes        types.Int64 `tfsdk:"idle_timeout_minutes"`

	Pitr              types.Bool  `tfsdk:"pitr"`
	PitrRetentionDays types.Int64 `tfsdk:"pitr_retention_days"`

	StorageAutoscaling                 types.Bool  `tfsdk:"storage_autoscaling"`
	StorageAutoscalingThresholdPercent types.Int64 `tfsdk:"storage_autoscaling_threshold_percent"`
	StorageAutoscalingMaxGb            types.Int64 `tfsdk:"storage_autoscaling_max_gb"`

	MaintenanceWindowDay     types.String `tfsdk:"maintenance_window_day"`
	MaintenanceWindowHourUTC types.Int64  `tfsdk:"maintenance_window_hour_utc"`

	SQLAPIEnabled           types.Bool  `tfsdk:"sql_api_enabled"`
	SQLAPIAllowedStatements types.Set   `tfsdk:"sql_api_allowed_statements"`
	SQLAPIMaxRows           types.Int64 `tfsdk:"sql_api_max_rows"`
	SQLAPIMaxBytes          types.Int64 `tfsdk:"sql_api_max_bytes"`
	SQLAPITimeoutSeconds    types.Int64 `tfsdk:"sql_api_timeout_seconds"`

	API                types.String `tfsdk:"api"`
	Engine             types.String `tfsdk:"engine"`
	Backend            types.String `tfsdk:"backend"`
	Hostname           types.String `tfsdk:"hostname"`
	ConnectionPort     types.Int64  `tfsdk:"connection_port"`
	ConnectionUser     types.String `tfsdk:"connection_user"`
	ConnectionPassword types.String `tfsdk:"connection_password"`
	ConnectionString   types.String `tfsdk:"connection_string"`
	SSL                types.Bool   `tfsdk:"ssl"`
	ContainerStatus    types.String `tfsdk:"container_status"`
	LastAccessedAt     types.String `tfsdk:"last_accessed_at"`
	IdleUntil          types.String `tfsdk:"idle_until"`
	LifecycleState     types.String `tfsdk:"lifecycle_state"`
	CPU                types.Int64  `tfsdk:"cpu"`
	Memory             types.Int64  `tfsdk:"memory"`
	Storage            types.Int64  `tfsdk:"storage"`
	StorageClass       types.String `tfsdk:"storage_class"`
	StorageMaxGb       types.Int64  `tfsdk:"storage_max_gb"`
	NodePool           types.String `tfsdk:"node_pool"`
	BackupEnabled      types.Bool   `tfsdk:"backup_enabled"`
	MetricsEnabled     types.Bool   `tfsdk:"metrics_enabled"`
	Error              types.String `tfsdk:"error"`

	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	ProjectID types.String `tfsdk:"project_id"`
}

// NewDatabaseDataSource returns a constructor for the dedicated database data
// source of one engine.
func NewDatabaseDataSource(engine Engine) func() datasource.DataSource {
	return func() datasource.DataSource {
		return &databaseDataSource{engine: engine}
	}
}

func (d *databaseDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_database", req.ProviderTypeName, d.engine)
}

func (d *databaseDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	computedString := func(description string) schema.StringAttribute {
		return schema.StringAttribute{Description: description, Computed: true}
	}
	computedInt := func(description string) schema.Int64Attribute {
		return schema.Int64Attribute{Description: description, Computed: true}
	}
	computedBool := func(description string) schema.BoolAttribute {
		return schema.BoolAttribute{Description: description, Computed: true}
	}
	computedStringSet := func(description string) schema.SetAttribute {
		return schema.SetAttribute{Description: description, Computed: true, ElementType: types.StringType}
	}

	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Fetches a dedicated Appwrite %s database by ID.", d.engine.Label()),
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

			"name":          computedString("The database display name."),
			"version":       computedString("The database engine version."),
			"specification": computedString("The compute specification slug."),
			"status":        computedString("The database status."),

			"replicas":  computedInt("The number of high availability replicas."),
			"sync_mode": computedString("The replication sync mode."),

			"network_idle_timeout_seconds": computedInt("The idle client connection timeout in seconds."),
			"network_ip_allowlist":         computedStringSet("The IP addresses and CIDR ranges allowed to connect."),
			"network_max_connections":      computedInt("The maximum number of concurrent client connections."),
			"idle_timeout_minutes":         computedInt("Minutes of inactivity before the container scales to zero."),

			"pitr":                computedBool("Whether point-in-time recovery is enabled."),
			"pitr_retention_days": computedInt("How many days of point-in-time recovery data are retained."),

			"storage_autoscaling":                   computedBool("Whether storage expands automatically."),
			"storage_autoscaling_threshold_percent": computedInt("The usage percentage that triggers automatic expansion."),
			"storage_autoscaling_max_gb":            computedInt("The storage ceiling in GB for autoscaling. 0 means no limit."),

			"maintenance_window_day":      computedString("The day of the week the maintenance window starts."),
			"maintenance_window_hour_utc": computedInt("The hour in UTC the maintenance window starts."),

			"sql_api_enabled":            computedBool("Whether the SQL API sidecar is enabled."),
			"sql_api_allowed_statements": computedStringSet("The statement types the SQL API accepts."),
			"sql_api_max_rows":           computedInt("The maximum rows returned per SQL API execution."),
			"sql_api_max_bytes":          computedInt("The maximum SQL API result payload in bytes."),
			"sql_api_timeout_seconds":    computedInt("The maximum SQL API execution time in seconds."),

			"api":              computedString("The product API that owns this database."),
			"engine":           computedString("The database engine reported by the server."),
			"backend":          computedString("The database backend provider."),
			"hostname":         computedString("The hostname to connect to."),
			"connection_port":  computedInt("The port to connect to."),
			"connection_user":  computedString("The username to connect with."),
			"ssl":              computedBool("Whether SSL/TLS is required for client connections."),
			"container_status": computedString("The container status for lifecycle-managed runtimes."),
			"last_accessed_at": computedString("The last activity timestamp in ISO 8601 format."),
			"idle_until":       computedString("When the database is expected to be considered idle."),
			"lifecycle_state":  computedString("The idle-lifecycle state."),
			"cpu":              computedInt("The allocated CPU in millicores."),
			"memory":           computedInt("The allocated memory in MB."),
			"storage":          computedInt("The allocated storage in GB."),
			"storage_class":    computedString("The storage class backing the volume."),
			"storage_max_gb":   computedInt("The maximum storage allowed in GB."),
			"node_pool":        computedString("The node pool the database is scheduled on."),
			"backup_enabled":   computedBool("Whether automatic backups are enabled."),
			"metrics_enabled":  computedBool("Whether metrics collection is enabled."),
			"error":            computedString("The error message reported when the status is `failed`."),
			"created_at":       computedString("The database creation timestamp in ISO 8601 format."),
			"updated_at":       computedString("The database last update timestamp in ISO 8601 format."),

			"connection_password": schema.StringAttribute{
				Description: "The password to connect with.",
				Computed:    true,
				Sensitive:   true,
			},
			"connection_string": schema.StringAttribute{
				Description: "The full connection URI, including credentials.",
				Computed:    true,
				Sensitive:   true,
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

	db, err := clientFor(d.clients, d.engine, projectID).Get(config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading dedicated database", common.FormatError(err))
		return
	}

	config.ProjectID = types.StringValue(projectID)
	config.ID = types.StringValue(db.Id)
	config.Name = types.StringValue(db.Name)
	config.Version = types.StringValue(db.Version)
	config.Specification = types.StringValue(db.Specification)
	config.Status = types.StringValue(db.Status)

	config.Replicas = types.Int64Value(int64(db.Replicas))
	config.SyncMode = types.StringValue(db.SyncMode)

	config.NetworkIdleTimeoutSeconds = types.Int64Value(int64(db.NetworkIdleTimeoutSeconds))
	config.NetworkMaxConnections = types.Int64Value(int64(db.NetworkMaxConnections))
	config.IdleTimeoutMinutes = types.Int64Value(int64(db.IdleTimeoutMinutes))

	config.Pitr = types.BoolValue(db.Pitr)
	config.PitrRetentionDays = types.Int64Value(int64(db.PitrRetentionDays))

	config.StorageAutoscaling = types.BoolValue(db.StorageAutoscaling)
	config.StorageAutoscalingThresholdPercent = types.Int64Value(int64(db.StorageAutoscalingThresholdPercent))
	config.StorageAutoscalingMaxGb = types.Int64Value(int64(db.StorageAutoscalingMaxGb))

	config.MaintenanceWindowDay = types.StringValue(db.MaintenanceWindowDay)
	config.MaintenanceWindowHourUTC = types.Int64Value(int64(db.MaintenanceWindowHourUtc))

	config.SQLAPIEnabled = types.BoolValue(db.SqlApiEnabled)
	config.SQLAPIMaxRows = types.Int64Value(int64(db.SqlApiMaxRows))
	config.SQLAPIMaxBytes = types.Int64Value(int64(db.SqlApiMaxBytes))
	config.SQLAPITimeoutSeconds = types.Int64Value(int64(db.SqlApiTimeoutSeconds))

	config.API = types.StringValue(db.Api)
	config.Engine = types.StringValue(db.Engine)
	config.Backend = types.StringValue(db.Backend)
	config.Hostname = types.StringValue(db.Hostname)
	config.ConnectionPort = types.Int64Value(int64(db.ConnectionPort))
	config.ConnectionUser = types.StringValue(db.ConnectionUser)
	config.ConnectionPassword = types.StringValue(db.ConnectionPassword)
	config.ConnectionString = types.StringValue(db.ConnectionString)
	config.SSL = types.BoolValue(db.Ssl)
	config.ContainerStatus = types.StringValue(db.ContainerStatus)
	config.LastAccessedAt = types.StringValue(db.LastAccessedAt)
	config.IdleUntil = types.StringValue(db.IdleUntil)
	config.LifecycleState = types.StringValue(db.LifecycleState)
	config.CPU = types.Int64Value(int64(db.Cpu))
	config.Memory = types.Int64Value(int64(db.Memory))
	config.Storage = types.Int64Value(int64(db.Storage))
	config.StorageClass = types.StringValue(db.StorageClass)
	config.StorageMaxGb = types.Int64Value(int64(db.StorageMaxGb))
	config.NodePool = types.StringValue(db.NodePool)
	config.BackupEnabled = types.BoolValue(db.BackupEnabled)
	config.MetricsEnabled = types.BoolValue(db.MetricsEnabled)
	config.Error = types.StringValue(db.Error)

	config.CreatedAt = types.StringValue(db.CreatedAt)
	config.UpdatedAt = types.StringValue(db.UpdatedAt)

	allowlist, diags := types.SetValueFrom(ctx, types.StringType, nonNilStrings(db.NetworkIPAllowlist))
	resp.Diagnostics.Append(diags...)
	config.NetworkIPAllowlist = allowlist

	statements, diags := types.SetValueFrom(ctx, types.StringType, nonNilStrings(db.SqlApiAllowedStatements))
	resp.Diagnostics.Append(diags...)
	config.SQLAPIAllowedStatements = statements

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
