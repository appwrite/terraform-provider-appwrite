package dedicated

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &poolerResource{}
	_ resource.ResourceWithConfigure   = &poolerResource{}
	_ resource.ResourceWithImportState = &poolerResource{}
)

// poolerResource manages the connection pooler that fronts a dedicated
// database. The pooler is part of the database rather than a separate object,
// so it is never created or destroyed -- creating this resource configures the
// existing pooler and destroying it only drops the settings from state.
type poolerResource struct {
	engine  Engine
	clients *common.AppwriteClients
}

type poolerResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	DatabaseID          types.String `tfsdk:"database_id"`
	Mode                types.String `tfsdk:"mode"`
	MaxConnections      types.Int64  `tfsdk:"max_connections"`
	DefaultPoolSize     types.Int64  `tfsdk:"default_pool_size"`
	ReadWriteSplitting  types.Bool   `tfsdk:"read_write_splitting"`
	PoolerCPURequest    types.String `tfsdk:"pooler_cpu_request"`
	PoolerCPULimit      types.String `tfsdk:"pooler_cpu_limit"`
	PoolerMemoryRequest types.String `tfsdk:"pooler_memory_request"`
	PoolerMemoryLimit   types.String `tfsdk:"pooler_memory_limit"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	Port                types.Int64  `tfsdk:"port"`
	ProjectID           types.String `tfsdk:"project_id"`
}

// NewPoolerResource returns a constructor for the pooler resource.
func NewPoolerResource(engine Engine) func() resource.Resource {
	return func() resource.Resource {
		return &poolerResource{engine: engine}
	}
}

func (r *poolerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_pooler", req.ProviderTypeName, r.engine)
}

func (r *poolerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	// The PostgreSQL pooler has no client-connection cap of its own: it reports
	// the database's network_max_connections and ignores anything sent. Marking
	// the attribute computed-only there makes Terraform reject a configured
	// value during planning, instead of the provider having to guess at apply
	// time whether a known value came from the user or from prior state.
	maxConnections := schema.Int64Attribute{
		Description:   "The client-connection ceiling the pooler accepts.",
		Optional:      true,
		Computed:      true,
		Validators:    []validator.Int64{int64validator.AtLeast(1)},
		PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
	}
	if r.engine == EnginePostgresql {
		maxConnections = schema.Int64Attribute{
			Description: "The client-connection ceiling the pooler accepts. Read-only on PostgreSQL, where the pooler has no client cap of its own and this reports the database's `network_max_connections` instead. Size it through the database's specification.",
			Computed:    true,
			// No UseStateForUnknown: this mirrors the database's
			// network_max_connections, which changes when the database is
			// resized, so the prior value is not a safe prediction.
		}
	}

	resp.Schema = schema.Schema{
		Description: fmt.Sprintf(
			"Configures the connection pooler of a dedicated Appwrite %s database. The pooler exists for the lifetime of the "+
				"database, so this resource only ever updates its settings: destroying it leaves the pooler running with its "+
				"last applied configuration.",
			r.engine.Label(),
		),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The pooler identifier, which is the database ID it belongs to.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"database_id": schema.StringAttribute{
				Description:   "The dedicated database ID whose pooler is configured.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"mode": schema.StringAttribute{
				Description:   "The pool mode. `transaction` returns a connection to the pool after each transaction; `session` holds it for the whole client session.",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.String{stringvalidator.OneOf("transaction", "session")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"max_connections": maxConnections,
			"default_pool_size": schema.Int64Attribute{
				Description:   "The default pool size per user.",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.Int64{int64validator.AtLeast(1)},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"read_write_splitting": schema.BoolAttribute{
				Description:   "Whether SELECTs are routed to high availability replicas while writes and locked reads stay on the primary. Only active when the database has replicas.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"pooler_cpu_request": schema.StringAttribute{
				Description:   "The CPU request for the pooler sidecar as a Kubernetes quantity, for example `100m`. Defaults to a proportion of the database CPU.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"pooler_cpu_limit": schema.StringAttribute{
				Description:   "The CPU limit for the pooler sidecar as a Kubernetes quantity, for example `200m`.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"pooler_memory_request": schema.StringAttribute{
				Description:   "The memory request for the pooler sidecar as a Kubernetes quantity, for example `64Mi`.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"pooler_memory_limit": schema.StringAttribute{
				Description:   "The memory limit for the pooler sidecar as a Kubernetes quantity, for example `128Mi`.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether connection pooling is enabled.",
				Computed:    true,
			},
			"port": schema.Int64Attribute{
				Description: "The port the pooler listens on.",
				Computed:    true,
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

func (r *poolerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *poolerResource) api(resourceProjectID types.String) (poolerAPI, string, error) {
	projectID, err := common.ResolveProjectID(r.clients, resourceProjectID)
	if err != nil {
		return nil, "", err
	}
	api, ok := newPoolerAPI(r.engine, r.clients.ClientForProject(projectID))
	if !ok {
		return nil, "", fmt.Errorf("%s databases do not have a connection pooler", r.engine.Label())
	}
	return api, projectID, nil
}

func (r *poolerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan poolerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, plan, &resp.Diagnostics, &resp.State)
}

func (r *poolerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan poolerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, plan, &resp.Diagnostics, &resp.State)
}

// apply pushes the planned pooler settings and writes the server's answer back
// to state. Create and Update are the same operation for a pooler.
func (r *poolerResource) apply(ctx context.Context, plan poolerResourceModel, diagnostics *diag.Diagnostics, state *tfsdk.State) {
	api, projectID, err := r.api(plan.ProjectID)
	if err != nil {
		diagnostics.AddError("Error resolving pooler client", err.Error())
		return
	}

	// On PostgreSQL the attribute is computed-only, so any value in the plan was
	// read back from the server rather than configured. Sending it would be a
	// no-op at best, so it is never forwarded.
	var maxConnections *int
	if r.engine != EnginePostgresql {
		maxConnections = optInt(plan.MaxConnections)
	}

	pooler, err := api.UpdatePooler(plan.DatabaseID.ValueString(), PoolerOptions{
		Mode:                optString(plan.Mode),
		MaxConnections:      maxConnections,
		DefaultPoolSize:     optInt(plan.DefaultPoolSize),
		ReadWriteSplitting:  optBool(plan.ReadWriteSplitting),
		PoolerCPURequest:    optString(plan.PoolerCPURequest),
		PoolerCPULimit:      optString(plan.PoolerCPULimit),
		PoolerMemoryRequest: optString(plan.PoolerMemoryRequest),
		PoolerMemoryLimit:   optString(plan.PoolerMemoryLimit),
	})
	if err != nil {
		diagnostics.AddError("Error updating connection pooler", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(pooler, &plan)
	diagnostics.Append(state.Set(ctx, &plan)...)
}

func (r *poolerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state poolerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	api, projectID, err := r.api(state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving pooler client", err.Error())
		return
	}

	pooler, err := api.GetPooler(state.DatabaseID.ValueString())
	if err != nil {
		// The pooler only exists while its database does.
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading connection pooler", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	r.mapToState(pooler, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state only. A pooler cannot be deleted
// independently of its database, and tearing down pooling on a live database
// would break every client connected through it.
func (r *poolerResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *poolerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *poolerResource) mapToState(pooler *models.DedicatedDatabasePooler, model *poolerResourceModel) {
	model.ID = types.StringValue(model.DatabaseID.ValueString())
	model.Mode = types.StringValue(pooler.Mode)
	model.MaxConnections = types.Int64Value(int64(pooler.MaxConnections))
	model.DefaultPoolSize = types.Int64Value(int64(pooler.DefaultPoolSize))
	model.ReadWriteSplitting = types.BoolValue(pooler.ReadWriteSplitting)
	model.PoolerCPURequest = types.StringValue(pooler.PoolerCpuRequest)
	model.PoolerCPULimit = types.StringValue(pooler.PoolerCpuLimit)
	model.PoolerMemoryRequest = types.StringValue(pooler.PoolerMemoryRequest)
	model.PoolerMemoryLimit = types.StringValue(pooler.PoolerMemoryLimit)
	model.Enabled = types.BoolValue(pooler.Enabled)
	model.Port = types.Int64Value(int64(pooler.Port))
}
