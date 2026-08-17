package dedicated

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/id"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                     = &databaseResource{}
	_ resource.ResourceWithConfigure        = &databaseResource{}
	_ resource.ResourceWithImportState      = &databaseResource{}
	_ resource.ResourceWithConfigValidators = &databaseResource{}
)

type databaseResource struct {
	engine  Engine
	clients *common.AppwriteClients
}

type databaseResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Version       types.String `tfsdk:"version"`
	Specification types.String `tfsdk:"specification"`
	Status        types.String `tfsdk:"status"`

	Replicas types.Int64  `tfsdk:"replicas"`
	SyncMode types.String `tfsdk:"sync_mode"`

	NetworkIdleTimeoutSeconds types.Int64 `tfsdk:"network_idle_timeout_seconds"`
	NetworkIPAllowlist        types.Set   `tfsdk:"network_ip_allowlist"`
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

	API                   types.String `tfsdk:"api"`
	Engine                types.String `tfsdk:"engine"`
	Backend               types.String `tfsdk:"backend"`
	Hostname              types.String `tfsdk:"hostname"`
	ConnectionPort        types.Int64  `tfsdk:"connection_port"`
	ConnectionUser        types.String `tfsdk:"connection_user"`
	ConnectionPassword    types.String `tfsdk:"connection_password"`
	ConnectionString      types.String `tfsdk:"connection_string"`
	SSL                   types.Bool   `tfsdk:"ssl"`
	ContainerStatus       types.String `tfsdk:"container_status"`
	LastAccessedAt        types.String `tfsdk:"last_accessed_at"`
	IdleUntil             types.String `tfsdk:"idle_until"`
	LifecycleState        types.String `tfsdk:"lifecycle_state"`
	CPU                   types.Int64  `tfsdk:"cpu"`
	Memory                types.Int64  `tfsdk:"memory"`
	Storage               types.Int64  `tfsdk:"storage"`
	StorageClass          types.String `tfsdk:"storage_class"`
	StorageMaxGb          types.Int64  `tfsdk:"storage_max_gb"`
	NodePool              types.String `tfsdk:"node_pool"`
	NetworkMaxConnections types.Int64  `tfsdk:"network_max_connections"`
	BackupEnabled         types.Bool   `tfsdk:"backup_enabled"`
	MetricsEnabled        types.Bool   `tfsdk:"metrics_enabled"`
	Error                 types.String `tfsdk:"error"`

	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	ProjectID types.String `tfsdk:"project_id"`
}

// NewDatabaseResource returns a constructor for the dedicated database resource
// of one engine.
func NewDatabaseResource(engine Engine) func() resource.Resource {
	return func() resource.Resource {
		return &databaseResource{engine: engine}
	}
}

func (r *databaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_database", req.ProviderTypeName, r.engine)
}

func (r *databaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	label := r.engine.Label()

	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description:   "The database ID. Must be unique within the project. Generated when omitted.",
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
		},
		"name": schema.StringAttribute{
			Description: "The database display name.",
			Required:    true,
		},
		"version": schema.StringAttribute{
			Description:   fmt.Sprintf("The %s engine version. Changing this performs an in-place major version upgrade, which cannot be rolled back.", label),
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"specification": schema.StringAttribute{
			Description:   "The compute specification slug, for example `db-s-1vcpu-1gb`. Read the available slugs from the corresponding specifications data source. Changing this resizes the database in place.",
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"status": schema.StringAttribute{
			Description: "The desired database status. Set to `paused` to stop the database without deleting it, and back to `ready` to resume it. Left unset, the status is only read back from the server.",
			Optional:    true,
			Computed:    true,
			// Deliberately no UseStateForUnknown: the server moves a database
			// between statuses on its own (idle timeouts, maintenance), so the
			// prior value is not a safe prediction.
			Validators: []validator.String{stringvalidator.OneOf("ready", "paused", "inactive")},
		},
		"replicas": schema.Int64Attribute{
			Description:   "The number of high availability replicas. High availability is enabled when greater than 0.",
			Optional:      true,
			Computed:      true,
			Validators:    []validator.Int64{int64validator.AtLeast(0)},
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},
		"sync_mode": schema.StringAttribute{
			Description:   "The replication sync mode. One of `async`, `sync` or `quorum`. Only meaningful when `replicas` is greater than 0.",
			Optional:      true,
			Computed:      true,
			Validators:    []validator.String{stringvalidator.OneOf("async", "sync", "quorum")},
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"network_idle_timeout_seconds": schema.Int64Attribute{
			Description:   "How long an idle client connection is held open before the server closes it.",
			Optional:      true,
			Computed:      true,
			Validators:    []validator.Int64{int64validator.AtLeast(0)},
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},
		"network_ip_allowlist": schema.SetAttribute{
			Description:   "IP addresses and CIDR ranges allowed to connect. An empty set allows any address.",
			Optional:      true,
			Computed:      true,
			ElementType:   types.StringType,
			PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
		},
		"idle_timeout_minutes": schema.Int64Attribute{
			Description:   "Minutes of inactivity before the database container scales to zero. Set to 0 to keep it always on.",
			Optional:      true,
			Computed:      true,
			Validators:    []validator.Int64{int64validator.AtLeast(0)},
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},
		"pitr": schema.BoolAttribute{
			Description:   "Whether point-in-time recovery is enabled.",
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
		},
		"pitr_retention_days": schema.Int64Attribute{
			Description:   "How many days of point-in-time recovery data to retain.",
			Optional:      true,
			Computed:      true,
			Validators:    []validator.Int64{int64validator.AtLeast(0)},
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},
		"storage_autoscaling": schema.BoolAttribute{
			Description:   "Whether storage expands automatically as it fills up.",
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
		},
		"storage_autoscaling_threshold_percent": schema.Int64Attribute{
			Description:   "The storage usage percentage that triggers automatic expansion.",
			Optional:      true,
			Computed:      true,
			Validators:    []validator.Int64{int64validator.Between(1, 100)},
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},
		"storage_autoscaling_max_gb": schema.Int64Attribute{
			Description:   "The storage ceiling in GB for autoscaling. 0 means no limit.",
			Optional:      true,
			Computed:      true,
			Validators:    []validator.Int64{int64validator.AtLeast(0)},
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},
		"maintenance_window_day": schema.StringAttribute{
			Description:   "The day of the week the maintenance window starts. One of `sun`, `mon`, `tue`, `wed`, `thu`, `fri` or `sat`. Must be set together with `maintenance_window_hour_utc`.",
			Optional:      true,
			Computed:      true,
			Validators:    []validator.String{stringvalidator.OneOf("sun", "mon", "tue", "wed", "thu", "fri", "sat")},
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"maintenance_window_hour_utc": schema.Int64Attribute{
			Description:   "The hour in UTC (0-23) the maintenance window starts. Must be set together with `maintenance_window_day`.",
			Optional:      true,
			Computed:      true,
			Validators:    []validator.Int64{int64validator.Between(0, 23)},
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},
		"sql_api_enabled": schema.BoolAttribute{
			Description:   "Whether the SQL API sidecar is enabled, allowing statements to be run over the Appwrite API.",
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
		},
		"sql_api_allowed_statements": schema.SetAttribute{
			Description:   "The statement types the SQL API accepts. Defaults to read/write DML only; DDL and DCL types (`CREATE`, `ALTER`, `DROP`, `TRUNCATE`, `GRANT`, `REVOKE`) are opt-in.",
			Optional:      true,
			Computed:      true,
			ElementType:   types.StringType,
			PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
		},
		"sql_api_max_rows": schema.Int64Attribute{
			Description:   "The maximum number of rows returned per SQL API execution. Larger results are truncated.",
			Optional:      true,
			Computed:      true,
			Validators:    []validator.Int64{int64validator.AtLeast(1)},
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},
		"sql_api_max_bytes": schema.Int64Attribute{
			Description:   "The maximum serialized SQL API result payload in bytes. Larger results are truncated.",
			Optional:      true,
			Computed:      true,
			Validators:    []validator.Int64{int64validator.AtLeast(1)},
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},
		"sql_api_timeout_seconds": schema.Int64Attribute{
			Description:   "The maximum server-side SQL API execution time in seconds before a query is canceled.",
			Optional:      true,
			Computed:      true,
			Validators:    []validator.Int64{int64validator.AtLeast(1)},
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},

		"api":                     schema.StringAttribute{Description: "The product API that owns this database.", Computed: true},
		"engine":                  schema.StringAttribute{Description: "The database engine reported by the server.", Computed: true},
		"backend":                 schema.StringAttribute{Description: "The database backend provider.", Computed: true},
		"hostname":                schema.StringAttribute{Description: "The hostname to connect to.", Computed: true},
		"connection_port":         schema.Int64Attribute{Description: "The port to connect to.", Computed: true},
		"connection_user":         schema.StringAttribute{Description: "The username to connect with.", Computed: true},
		"container_status":        schema.StringAttribute{Description: "The container status for lifecycle-managed runtimes.", Computed: true},
		"last_accessed_at":        schema.StringAttribute{Description: "The last activity timestamp in ISO 8601 format.", Computed: true},
		"idle_until":              schema.StringAttribute{Description: "When the database is expected to be considered idle, in ISO 8601 format.", Computed: true},
		"lifecycle_state":         schema.StringAttribute{Description: "The idle-lifecycle state: `active`, `warm`, `cold` or `hibernated`.", Computed: true},
		"cpu":                     schema.Int64Attribute{Description: "The allocated CPU in millicores.", Computed: true},
		"memory":                  schema.Int64Attribute{Description: "The allocated memory in MB.", Computed: true},
		"storage":                 schema.Int64Attribute{Description: "The allocated storage in GB.", Computed: true},
		"storage_class":           schema.StringAttribute{Description: "The storage class backing the volume.", Computed: true},
		"storage_max_gb":          schema.Int64Attribute{Description: "The maximum storage allowed in GB. 0 means the system default.", Computed: true},
		"node_pool":               schema.StringAttribute{Description: "The node pool the database is scheduled on.", Computed: true},
		"network_max_connections": schema.Int64Attribute{Description: "The maximum number of concurrent client connections.", Computed: true},
		"backup_enabled":          schema.BoolAttribute{Description: "Whether automatic backups are enabled.", Computed: true},
		"metrics_enabled":         schema.BoolAttribute{Description: "Whether metrics collection is enabled.", Computed: true},
		"error":                   schema.StringAttribute{Description: "The error message reported by the server when the status is `failed`.", Computed: true},
		"ssl":                     schema.BoolAttribute{Description: "Whether SSL/TLS is required for client connections.", Computed: true},

		"connection_password": schema.StringAttribute{
			Description: "The password to connect with. Rotate it out of band; this provider only reads it.",
			Computed:    true,
			Sensitive:   true,
		},
		"connection_string": schema.StringAttribute{
			Description: "The full connection URI, including credentials.",
			Computed:    true,
			Sensitive:   true,
		},

		"created_at": schema.StringAttribute{
			Description: "The database creation timestamp in ISO 8601 format.",
			Computed:    true,
		},
		"updated_at": schema.StringAttribute{
			Description:   "The database last update timestamp in ISO 8601 format.",
			Computed:      true,
			PlanModifiers: []planmodifier.String{common.UseStateForUnknownUnlessUpdating()},
		},
		"project_id": common.ProjectIDAttribute(),
	}

	resp.Schema = schema.Schema{
		Description: fmt.Sprintf(
			"Manages a dedicated Appwrite %s database. A dedicated database runs on infrastructure reserved for one project, "+
				"so creating, resizing or upgrading one takes several minutes; Terraform waits for the database to settle before continuing.",
			label,
		),
		Attributes: attributes,
	}
}

// ConfigValidators enforces the day/hour pairing against the configuration
// rather than the plan. The plan is not usable for this: both attributes are
// optional-and-computed, so after the first apply UseStateForUnknown carries
// the server's values forward and every plan looks fully configured.
func (r *databaseResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.RequiredTogether(
			path.MatchRoot("maintenance_window_day"),
			path.MatchRoot("maintenance_window_hour_utc"),
		),
	}
}

func (r *databaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData),
		)
		return
	}
	r.clients = clients
}

func (r *databaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan databaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	api := clientFor(r.clients, r.engine, projectID)

	databaseID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		databaseID = id.Unique()
	}

	allowlist, diags := stringSlice(ctx, plan.NetworkIPAllowlist)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	db, err := api.Create(databaseID, plan.Name.ValueString(), CreateOptions{
		Version:                            optString(plan.Version),
		Specification:                      optString(plan.Specification),
		Replicas:                           optInt(plan.Replicas),
		SyncMode:                           optString(plan.SyncMode),
		NetworkIdleTimeoutSeconds:          optInt(plan.NetworkIdleTimeoutSeconds),
		NetworkIPAllowlist:                 allowlist,
		IdleTimeoutMinutes:                 optInt(plan.IdleTimeoutMinutes),
		Pitr:                               optBool(plan.Pitr),
		PitrRetentionDays:                  optInt(plan.PitrRetentionDays),
		StorageAutoscaling:                 optBool(plan.StorageAutoscaling),
		StorageAutoscalingThresholdPercent: optInt(plan.StorageAutoscalingThresholdPercent),
		StorageAutoscalingMaxGb:            optInt(plan.StorageAutoscalingMaxGb),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating dedicated database", common.FormatError(err))
		return
	}

	// The database is still provisioning when Create returns. Persist the ID
	// first so a failure past this point leaves a resource Terraform can clean
	// up rather than an orphan billing away in the project.
	plan.ID = types.StringValue(db.Id)
	plan.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), db.Id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectID)...)

	db, err = waitForStable(ctx, func() (*models.DedicatedDatabase, error) { return api.Get(db.Id) }, db.Id, createStableTimeout)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for dedicated database", err.Error())
		return
	}

	// Several attributes have no create-time equivalent, so they are applied
	// with a follow-up update. Without this the server would report defaults
	// that contradict the plan.
	db, err = r.applyPostCreateSettings(ctx, api, db, plan, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Error applying dedicated database settings", common.FormatError(err))
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	r.mapToState(ctx, db, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// applyPostCreateSettings sends the settings the create route does not accept.
func (r *databaseResource) applyPostCreateSettings(ctx context.Context, api databaseAPI, db *models.DedicatedDatabase, plan databaseResourceModel, diagnostics *diag.Diagnostics) (*models.DedicatedDatabase, error) {
	statements, diags := stringSlice(ctx, plan.SQLAPIAllowedStatements)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return db, nil
	}

	opts := UpdateOptions{
		Status:                  optString(plan.Status),
		SQLAPIEnabled:           optBool(plan.SQLAPIEnabled),
		SQLAPIAllowedStatements: statements,
		SQLAPIMaxRows:           optInt(plan.SQLAPIMaxRows),
		SQLAPIMaxBytes:          optInt(plan.SQLAPIMaxBytes),
		SQLAPITimeoutSeconds:    optInt(plan.SQLAPITimeoutSeconds),
	}
	configured := opts.Status != nil ||
		opts.SQLAPIEnabled != nil ||
		opts.SQLAPIAllowedStatements != nil ||
		opts.SQLAPIMaxRows != nil ||
		opts.SQLAPIMaxBytes != nil ||
		opts.SQLAPITimeoutSeconds != nil
	if configured {
		updated, err := api.Update(db.Id, opts)
		if err != nil {
			return db, err
		}
		db = updated
	}

	updated, err := r.applyMaintenanceWindow(api, db, plan, diagnostics)
	if err != nil {
		return db, err
	}
	return updated, nil
}

// applyMaintenanceWindow sets the maintenance window, which lives behind its
// own route rather than the general update.
func (r *databaseResource) applyMaintenanceWindow(api databaseAPI, db *models.DedicatedDatabase, plan databaseResourceModel, diagnostics *diag.Diagnostics) (*models.DedicatedDatabase, error) {
	day := optString(plan.MaintenanceWindowDay)
	hour := optInt(plan.MaintenanceWindowHourUTC)
	if day == nil && hour == nil {
		return db, nil
	}
	if day == nil || hour == nil {
		diagnostics.AddError(
			"Incomplete maintenance window",
			"maintenance_window_day and maintenance_window_hour_utc must be set together; the API takes the day and hour as one pair.",
		)
		return db, nil
	}
	if db.MaintenanceWindowDay == *day && db.MaintenanceWindowHourUtc == *hour {
		return db, nil
	}
	return api.UpdateMaintenance(db.Id, *day, *hour)
}

func (r *databaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state databaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	api := clientFor(r.clients, r.engine, projectID)

	db, err := api.Get(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading dedicated database", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, db, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *databaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state databaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	api := clientFor(r.clients, r.engine, projectID)
	databaseID := state.ID.ValueString()

	// A version change is an upgrade, not a field update, and has to go first
	// so the rest of the settings land on the upgraded engine.
	if v := optString(plan.Version); v != nil && *v != state.Version.ValueString() {
		if _, err := api.CreateUpgrade(databaseID, *v); err != nil {
			resp.Diagnostics.AddError("Error upgrading dedicated database", common.FormatError(err))
			return
		}
		if _, err := waitForStable(ctx, func() (*models.DedicatedDatabase, error) { return api.Get(databaseID) }, databaseID, updateStableTimeout); err != nil {
			resp.Diagnostics.AddError("Error waiting for dedicated database upgrade", err.Error())
			return
		}
	}

	allowlist, diags := stringSlice(ctx, plan.NetworkIPAllowlist)
	resp.Diagnostics.Append(diags...)
	statements, diags := stringSlice(ctx, plan.SQLAPIAllowedStatements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := api.Update(databaseID, UpdateOptions{
		Name:                               optString(plan.Name),
		Status:                             optString(plan.Status),
		Specification:                      optString(plan.Specification),
		Replicas:                           optInt(plan.Replicas),
		SyncMode:                           optString(plan.SyncMode),
		NetworkIdleTimeoutSeconds:          optInt(plan.NetworkIdleTimeoutSeconds),
		NetworkIPAllowlist:                 allowlist,
		IdleTimeoutMinutes:                 optInt(plan.IdleTimeoutMinutes),
		Pitr:                               optBool(plan.Pitr),
		PitrRetentionDays:                  optInt(plan.PitrRetentionDays),
		StorageAutoscaling:                 optBool(plan.StorageAutoscaling),
		StorageAutoscalingThresholdPercent: optInt(plan.StorageAutoscalingThresholdPercent),
		StorageAutoscalingMaxGb:            optInt(plan.StorageAutoscalingMaxGb),
		SQLAPIEnabled:                      optBool(plan.SQLAPIEnabled),
		SQLAPIAllowedStatements:            statements,
		SQLAPIMaxRows:                      optInt(plan.SQLAPIMaxRows),
		SQLAPIMaxBytes:                     optInt(plan.SQLAPIMaxBytes),
		SQLAPITimeoutSeconds:               optInt(plan.SQLAPITimeoutSeconds),
	}); err != nil {
		resp.Diagnostics.AddError("Error updating dedicated database", common.FormatError(err))
		return
	}

	// Resizes and replica changes put the database back into a transitional
	// status, so settle again before reporting the result.
	db, err := waitForStable(ctx, func() (*models.DedicatedDatabase, error) { return api.Get(databaseID) }, databaseID, updateStableTimeout)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for dedicated database update", err.Error())
		return
	}

	db, err = r.applyMaintenanceWindow(api, db, plan, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Error updating maintenance window", common.FormatError(err))
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, db, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *databaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state databaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	api := clientFor(r.clients, r.engine, projectID)

	if err := api.Delete(state.ID.ValueString()); err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting dedicated database", common.FormatError(err))
	}
}

func (r *databaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *databaseResource) mapToState(ctx context.Context, db *models.DedicatedDatabase, model *databaseResourceModel, diagnostics *diag.Diagnostics) {
	model.ID = types.StringValue(db.Id)
	model.Name = types.StringValue(db.Name)
	model.Version = types.StringValue(db.Version)
	model.Specification = types.StringValue(db.Specification)
	model.Status = types.StringValue(db.Status)

	model.Replicas = types.Int64Value(int64(db.Replicas))
	model.SyncMode = types.StringValue(db.SyncMode)

	model.NetworkIdleTimeoutSeconds = types.Int64Value(int64(db.NetworkIdleTimeoutSeconds))
	model.IdleTimeoutMinutes = types.Int64Value(int64(db.IdleTimeoutMinutes))

	model.Pitr = types.BoolValue(db.Pitr)
	model.PitrRetentionDays = types.Int64Value(int64(db.PitrRetentionDays))

	model.StorageAutoscaling = types.BoolValue(db.StorageAutoscaling)
	model.StorageAutoscalingThresholdPercent = types.Int64Value(int64(db.StorageAutoscalingThresholdPercent))
	model.StorageAutoscalingMaxGb = types.Int64Value(int64(db.StorageAutoscalingMaxGb))

	model.MaintenanceWindowDay = types.StringValue(db.MaintenanceWindowDay)
	model.MaintenanceWindowHourUTC = types.Int64Value(int64(db.MaintenanceWindowHourUtc))

	model.SQLAPIEnabled = types.BoolValue(db.SqlApiEnabled)
	model.SQLAPIMaxRows = types.Int64Value(int64(db.SqlApiMaxRows))
	model.SQLAPIMaxBytes = types.Int64Value(int64(db.SqlApiMaxBytes))
	model.SQLAPITimeoutSeconds = types.Int64Value(int64(db.SqlApiTimeoutSeconds))

	model.API = types.StringValue(db.Api)
	model.Engine = types.StringValue(db.Engine)
	model.Backend = types.StringValue(db.Backend)
	model.Hostname = types.StringValue(db.Hostname)
	model.ConnectionPort = types.Int64Value(int64(db.ConnectionPort))
	model.ConnectionUser = types.StringValue(db.ConnectionUser)
	model.ConnectionPassword = types.StringValue(db.ConnectionPassword)
	model.ConnectionString = types.StringValue(db.ConnectionString)
	model.SSL = types.BoolValue(db.Ssl)
	model.ContainerStatus = types.StringValue(db.ContainerStatus)
	model.LastAccessedAt = types.StringValue(db.LastAccessedAt)
	model.IdleUntil = types.StringValue(db.IdleUntil)
	model.LifecycleState = types.StringValue(db.LifecycleState)
	model.CPU = types.Int64Value(int64(db.Cpu))
	model.Memory = types.Int64Value(int64(db.Memory))
	model.Storage = types.Int64Value(int64(db.Storage))
	model.StorageClass = types.StringValue(db.StorageClass)
	model.StorageMaxGb = types.Int64Value(int64(db.StorageMaxGb))
	model.NodePool = types.StringValue(db.NodePool)
	model.NetworkMaxConnections = types.Int64Value(int64(db.NetworkMaxConnections))
	model.BackupEnabled = types.BoolValue(db.BackupEnabled)
	model.MetricsEnabled = types.BoolValue(db.MetricsEnabled)
	model.Error = types.StringValue(db.Error)

	model.CreatedAt = types.StringValue(db.CreatedAt)
	model.UpdatedAt = types.StringValue(db.UpdatedAt)

	allowlist, diags := types.SetValueFrom(ctx, types.StringType, nonNilStrings(db.NetworkIPAllowlist))
	diagnostics.Append(diags...)
	model.NetworkIPAllowlist = allowlist

	statements, diags := types.SetValueFrom(ctx, types.StringType, nonNilStrings(db.SqlApiAllowedStatements))
	diagnostics.Append(diags...)
	model.SQLAPIAllowedStatements = statements
}

// The framework distinguishes a null set from an empty one, while the API
// returns an omitted list as nil. Normalising to empty keeps a database with no
// allowlist from flipping between null and [] on refresh.
func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func optString(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

func optInt(v types.Int64) *int {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	i := int(v.ValueInt64())
	return &i
}

func optBool(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

func stringSlice(ctx context.Context, v types.Set) ([]string, diag.Diagnostics) {
	if v.IsNull() || v.IsUnknown() {
		return nil, nil
	}
	var out []string
	diags := v.ElementsAs(ctx, &out, false)
	if out == nil {
		out = []string{}
	}
	return out, diags
}
