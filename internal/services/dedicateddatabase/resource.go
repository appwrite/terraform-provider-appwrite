package dedicateddatabase

import (
	"context"
	"fmt"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/aw-tests/sdk-for-go/v6/id"
	"github.com/aw-tests/sdk-for-go/v6/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &databaseResource{}
	_ resource.ResourceWithConfigure   = &databaseResource{}
	_ resource.ResourceWithImportState = &databaseResource{}
)

type databaseResource struct {
	clients *common.AppwriteClients
}

type databaseResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Engine        types.String `tfsdk:"engine"`
	Name          types.String `tfsdk:"name"`
	Version       types.String `tfsdk:"version"`
	Specification types.String `tfsdk:"specification"`
	API           types.String `tfsdk:"api"`

	Replicas                           types.Int64  `tfsdk:"replicas"`
	SyncMode                           types.String `tfsdk:"sync_mode"`
	NetworkIdleTimeoutSeconds          types.Int64  `tfsdk:"network_idle_timeout_seconds"`
	NetworkIPAllowlist                 types.List   `tfsdk:"network_ip_allowlist"`
	IdleTimeoutMinutes                 types.Int64  `tfsdk:"idle_timeout_minutes"`
	Pitr                               types.Bool   `tfsdk:"pitr"`
	PitrRetentionDays                  types.Int64  `tfsdk:"pitr_retention_days"`
	StorageAutoscaling                 types.Bool   `tfsdk:"storage_autoscaling"`
	StorageAutoscalingThresholdPercent types.Int64  `tfsdk:"storage_autoscaling_threshold_percent"`
	StorageAutoscalingMaxGb            types.Int64  `tfsdk:"storage_autoscaling_max_gb"`

	Backend            types.String `tfsdk:"backend"`
	Hostname           types.String `tfsdk:"hostname"`
	ConnectionPort     types.Int64  `tfsdk:"connection_port"`
	ConnectionUser     types.String `tfsdk:"connection_user"`
	ConnectionPassword types.String `tfsdk:"connection_password"`
	ConnectionString   types.String `tfsdk:"connection_string"`
	Ssl                types.Bool   `tfsdk:"ssl"`
	Status             types.String `tfsdk:"status"`
	CPU                types.Int64  `tfsdk:"cpu"`
	Memory             types.Int64  `tfsdk:"memory"`
	Storage            types.Int64  `tfsdk:"storage"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
	ProjectID          types.String `tfsdk:"project_id"`
}

func NewDatabaseResource() resource.Resource {
	return &databaseResource{}
}

func (r *databaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_database"
}

func (r *databaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	optionalComputedInt := func(desc string) schema.Int64Attribute {
		return schema.Int64Attribute{Description: desc, Optional: true, Computed: true}
	}
	optionalComputedBool := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{Description: desc, Optional: true, Computed: true}
	}

	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite dedicated database (a managed PostgreSQL, MySQL, or MongoDB instance).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The dedicated database ID. Must be unique within the project.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"engine": schema.StringAttribute{
				Description:   fmt.Sprintf("The database engine. One of: %s.", validEngines()),
				Required:      true,
				Validators:    []validator.String{stringvalidator.OneOf("postgresql", "mysql", "mongo")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description: "The database display name.",
				Required:    true,
			},
			"version": schema.StringAttribute{
				Description:   "The engine version. Changing this forces a new database; use the Appwrite console to perform an in-place upgrade.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"specification": schema.StringAttribute{
				Description: "The compute specification identifier (e.g. a size tier). See the engine's list-specifications API for valid values.",
				Optional:    true,
				Computed:    true,
			},
			"api": schema.StringAttribute{
				Description:   "The product API that owns this database: nativedb, documentsdb, or vectorsdb. Changing this forces a new database.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},

			"replicas":                              optionalComputedInt("Number of high-availability replicas. High availability is enabled when greater than 0."),
			"sync_mode":                             schema.StringAttribute{Description: "Replication sync mode: async, sync, or quorum.", Optional: true, Computed: true},
			"network_idle_timeout_seconds":          optionalComputedInt("Connection idle timeout in seconds."),
			"network_ip_allowlist":                  schema.ListAttribute{Description: "IP addresses/CIDR ranges allowed to connect.", Optional: true, Computed: true, ElementType: types.StringType},
			"idle_timeout_minutes":                  optionalComputedInt("Minutes of inactivity before the container scales to zero."),
			"pitr":                                  optionalComputedBool("Whether point-in-time recovery is enabled."),
			"pitr_retention_days":                   optionalComputedInt("Number of days to retain point-in-time-recovery data."),
			"storage_autoscaling":                   optionalComputedBool("Whether automatic storage expansion is enabled."),
			"storage_autoscaling_threshold_percent": optionalComputedInt("Storage usage percentage that triggers automatic expansion."),
			"storage_autoscaling_max_gb":            optionalComputedInt("Maximum storage size in GB for autoscaling. 0 means no limit."),

			"backend":             schema.StringAttribute{Description: "Database backend provider (prisma or edge).", Computed: true},
			"hostname":            schema.StringAttribute{Description: "Database hostname for connections.", Computed: true},
			"connection_port":     schema.Int64Attribute{Description: "Database port for connections.", Computed: true},
			"connection_user":     schema.StringAttribute{Description: "Database username for connections.", Computed: true},
			"connection_password": schema.StringAttribute{Description: "Database password for connections.", Computed: true, Sensitive: true},
			"connection_string":   schema.StringAttribute{Description: "Full database connection string (URI format).", Computed: true, Sensitive: true},
			"ssl":                 schema.BoolAttribute{Description: "Whether SSL/TLS is required for client connections.", Computed: true},
			"status":              schema.StringAttribute{Description: "Database status (e.g. provisioning, ready, paused, failed).", Computed: true},
			"cpu":                 schema.Int64Attribute{Description: "CPU allocated in millicores.", Computed: true},
			"memory":              schema.Int64Attribute{Description: "Memory allocated in MB.", Computed: true},
			"storage":             schema.Int64Attribute{Description: "Storage allocated in GB.", Computed: true},
			"created_at":          schema.StringAttribute{Description: "The database creation timestamp in ISO 8601 format.", Computed: true},
			"updated_at": schema.StringAttribute{
				Description:   "The database last update timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{common.UseStateForUnknownUnlessUpdating()},
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

func (r *databaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData))
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
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	c := engineClient(r.clients, projectID)
	engine := plan.Engine.ValueString()

	dbID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		dbID = id.Unique()
	}

	params := map[string]interface{}{
		"databaseId": dbID,
		"name":       plan.Name.ValueString(),
	}
	setStr(params, "version", plan.Version)
	setStr(params, "specification", plan.Specification)
	setStr(params, "api", plan.API)
	setStr(params, "syncMode", plan.SyncMode)
	setInt(params, "replicas", plan.Replicas)
	setInt(params, "networkIdleTimeoutSeconds", plan.NetworkIdleTimeoutSeconds)
	setInt(params, "idleTimeoutMinutes", plan.IdleTimeoutMinutes)
	setBool(params, "pitr", plan.Pitr)
	setInt(params, "pitrRetentionDays", plan.PitrRetentionDays)
	setBool(params, "storageAutoscaling", plan.StorageAutoscaling)
	setInt(params, "storageAutoscalingThresholdPercent", plan.StorageAutoscalingThresholdPercent)
	setInt(params, "storageAutoscalingMaxGb", plan.StorageAutoscalingMaxGb)
	if allow, ok := r.stringList(ctx, plan.NetworkIPAllowlist, &resp.Diagnostics); ok {
		params["networkIPAllowlist"] = allow
	}
	if resp.Diagnostics.HasError() {
		return
	}

	var db models.DedicatedDatabase
	if err := apiCall(c, r.clients.UserAgent, "POST", "/"+engine, params, &db); err != nil {
		resp.Diagnostics.AddError("Error creating dedicated database", common.FormatError(err))
		return
	}

	// Provisioning is asynchronous; wait so connection details are populated.
	if err := waitForDatabaseReady(ctx, func() (string, error) {
		var cur models.DedicatedDatabase
		if err := apiCall(c, r.clients.UserAgent, "GET", "/"+engine+"/"+db.Id, nil, &cur); err != nil {
			return "", err
		}
		return cur.Status, nil
	}, db.Id); err != nil {
		resp.Diagnostics.AddError("Error waiting for dedicated database", err.Error())
		return
	}

	var ready models.DedicatedDatabase
	if err := apiCall(c, r.clients.UserAgent, "GET", "/"+engine+"/"+db.Id, nil, &ready); err != nil {
		resp.Diagnostics.AddError("Error reading dedicated database after create", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, &ready, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *databaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state databaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	c := engineClient(r.clients, projectID)
	engine := state.Engine.ValueString()

	var db models.DedicatedDatabase
	if err := apiCall(c, r.clients.UserAgent, "GET", "/"+engine+"/"+state.ID.ValueString(), nil, &db); err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading dedicated database", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, &db, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *databaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan databaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	c := engineClient(r.clients, projectID)
	engine := plan.Engine.ValueString()

	params := map[string]interface{}{"name": plan.Name.ValueString()}
	setStr(params, "specification", plan.Specification)
	setStr(params, "syncMode", plan.SyncMode)
	setInt(params, "replicas", plan.Replicas)
	setInt(params, "networkIdleTimeoutSeconds", plan.NetworkIdleTimeoutSeconds)
	setInt(params, "idleTimeoutMinutes", plan.IdleTimeoutMinutes)
	setBool(params, "pitr", plan.Pitr)
	setInt(params, "pitrRetentionDays", plan.PitrRetentionDays)
	setBool(params, "storageAutoscaling", plan.StorageAutoscaling)
	setInt(params, "storageAutoscalingThresholdPercent", plan.StorageAutoscalingThresholdPercent)
	setInt(params, "storageAutoscalingMaxGb", plan.StorageAutoscalingMaxGb)
	if allow, ok := r.stringList(ctx, plan.NetworkIPAllowlist, &resp.Diagnostics); ok {
		params["networkIPAllowlist"] = allow
	}
	if resp.Diagnostics.HasError() {
		return
	}

	var db models.DedicatedDatabase
	if err := apiCall(c, r.clients.UserAgent, "PATCH", "/"+engine+"/"+plan.ID.ValueString(), params, &db); err != nil {
		resp.Diagnostics.AddError("Error updating dedicated database", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, &db, &plan, &resp.Diagnostics)
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
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	c := engineClient(r.clients, projectID)
	engine := state.Engine.ValueString()

	if err := apiCall[any](c, r.clients.UserAgent, "DELETE", "/"+engine+"/"+state.ID.ValueString(), nil, nil); err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting dedicated database", common.FormatError(err))
	}
}

// ImportState expects "engine/database_id" since the engine selects the API endpoint.
func (r *databaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	engine, dbID, ok := splitTwo(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: engine/database_id (e.g. postgresql/abc123), got: %s", req.ID))
		return
	}
	if _, valid := engines[engine]; !valid {
		resp.Diagnostics.AddError("Invalid engine", fmt.Sprintf("Engine must be one of: %s", validEngines()))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("engine"), engine)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), dbID)...)
}

func (r *databaseResource) stringList(ctx context.Context, list types.List, diags *diag.Diagnostics) ([]string, bool) {
	if list.IsNull() || list.IsUnknown() {
		return nil, false
	}
	var out []string
	diags.Append(list.ElementsAs(ctx, &out, false)...)
	return out, true
}

func (r *databaseResource) mapToState(ctx context.Context, db *models.DedicatedDatabase, model *databaseResourceModel, diags *diag.Diagnostics) {
	// engine is intentionally not overwritten: it is the config-supplied endpoint
	// selector (postgresql/mysql/mongo), which may differ from the API's engine
	// field (e.g. "mongodb").
	model.ID = types.StringValue(db.Id)
	model.Name = types.StringValue(db.Name)
	model.Version = types.StringValue(db.Version)
	model.Specification = types.StringValue(db.Specification)
	model.API = types.StringValue(db.Api)

	model.Replicas = types.Int64Value(int64(db.Replicas))
	model.SyncMode = types.StringValue(db.SyncMode)
	model.NetworkIdleTimeoutSeconds = types.Int64Value(int64(db.NetworkIdleTimeoutSeconds))
	model.IdleTimeoutMinutes = types.Int64Value(int64(db.IdleTimeoutMinutes))
	model.Pitr = types.BoolValue(db.Pitr)
	model.PitrRetentionDays = types.Int64Value(int64(db.PitrRetentionDays))
	model.StorageAutoscaling = types.BoolValue(db.StorageAutoscaling)
	model.StorageAutoscalingThresholdPercent = types.Int64Value(int64(db.StorageAutoscalingThresholdPercent))
	model.StorageAutoscalingMaxGb = types.Int64Value(int64(db.StorageAutoscalingMaxGb))

	allow, d := types.ListValueFrom(ctx, types.StringType, db.NetworkIPAllowlist)
	diags.Append(d...)
	model.NetworkIPAllowlist = allow

	model.Backend = types.StringValue(db.Backend)
	model.Hostname = types.StringValue(db.Hostname)
	model.ConnectionPort = types.Int64Value(int64(db.ConnectionPort))
	model.ConnectionUser = types.StringValue(db.ConnectionUser)
	model.ConnectionPassword = types.StringValue(db.ConnectionPassword)
	model.ConnectionString = types.StringValue(db.ConnectionString)
	model.Ssl = types.BoolValue(db.Ssl)
	model.Status = types.StringValue(db.Status)
	model.CPU = types.Int64Value(int64(db.Cpu))
	model.Memory = types.Int64Value(int64(db.Memory))
	model.Storage = types.Int64Value(int64(db.Storage))
	model.CreatedAt = types.StringValue(db.CreatedAt)
	model.UpdatedAt = types.StringValue(db.UpdatedAt)
}
