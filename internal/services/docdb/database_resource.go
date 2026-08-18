package docdb

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/id"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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
	product Product
	clients *common.AppwriteClients
}

type databaseResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	Specification types.String `tfsdk:"specification"`
	Replicas      types.Int64  `tfsdk:"replicas"`
	SyncMode      types.String `tfsdk:"sync_mode"`

	Type      types.String `tfsdk:"type"`
	Status    types.String `tfsdk:"status"`
	Engine    types.String `tfsdk:"engine"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	ProjectID types.String `tfsdk:"project_id"`
}

// NewDatabaseResource returns a constructor for the database resource of one
// product.
func NewDatabaseResource(product Product) func() resource.Resource {
	return func() resource.Resource {
		return &databaseResource{product: product}
	}
}

func (r *databaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s", req.ProviderTypeName, r.product)
}

func (r *databaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	label := r.product.Label()

	description := fmt.Sprintf("Manages an Appwrite %s database.", label)
	if r.product == ProductVectorsDB {
		description += " Collections in a VectorsDB database store embeddings and are searched by vector similarity."
	} else {
		description += " Collections in a DocumentsDB database store schemaless JSON documents."
	}
	description += "\n\nSetting `specification` places the database on dedicated infrastructure reserved for this project, which is billed separately. " +
		"Left unset, it runs on the deployment's shared pool -- but not every deployment has one. Where none is configured the API rejects " +
		"creation with `dedicated_database_required`, and a `specification` becomes mandatory. Read the available slugs from the matching " +
		"specifications data source."

	resp.Schema = schema.Schema{
		Description: description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The database ID. Must be unique within the project. Generated when omitted.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The database name.",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the database is enabled. When disabled it is inaccessible to users but still reachable with an API key. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"specification": schema.StringAttribute{
				Description:   "The compute specification slug of the dedicated backing, for example `s-1vcpu-1gb`. Omit to use the deployment's shared pool, which requires that one is configured; otherwise this is mandatory.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"replicas": schema.Int64Attribute{
				Description:   "The number of high availability replicas on the dedicated backing, excluding the primary.",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.Int64{int64validator.AtLeast(0)},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"sync_mode": schema.StringAttribute{
				Description:   "The replication sync mode of the dedicated backing. One of `async`, `sync` or `quorum`. Only meaningful when `replicas` is greater than 0.",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.String{stringvalidator.OneOf("async", "sync", "quorum")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"type":   schema.StringAttribute{Description: "The database type reported by the server.", Computed: true},
			"status": schema.StringAttribute{Description: "The dedicated backing's lifecycle status. Empty when the database has no dedicated backing.", Computed: true},
			"engine": schema.StringAttribute{Description: "The engine the dedicated backing runs on. Empty when the database has no dedicated backing.", Computed: true},

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
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	api := apiFor(r.clients, r.product, projectID)

	databaseID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		databaseID = id.Unique()
	}

	database, err := api.Create(databaseID, plan.Name.ValueString(), DatabaseOptions{
		Enabled:       optBool(plan.Enabled),
		Specification: optString(plan.Specification),
		Replicas:      optInt(plan.Replicas),
		SyncMode:      optString(plan.SyncMode),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating database", common.FormatError(err))
		return
	}

	// A database with a dedicated backing is still provisioning when Create
	// returns. Persist the ID first so a failure past this point leaves a
	// resource Terraform can clean up rather than an orphan that keeps billing.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), database.Id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectID)...)

	database, err = waitForReady(ctx, func() (*models.Database, error) { return api.Get(database.Id) }, database.Id)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for database", err.Error())
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(database, &plan)
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
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}

	database, err := apiFor(r.clients, r.product, projectID).Get(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading database", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	r.mapToState(database, &state)
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
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}

	database, err := apiFor(r.clients, r.product, projectID).Update(plan.ID.ValueString(), plan.Name.ValueString(), DatabaseOptions{
		Enabled:       optBool(plan.Enabled),
		Specification: optString(plan.Specification),
		Replicas:      optInt(plan.Replicas),
		SyncMode:      optString(plan.SyncMode),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating database", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(database, &plan)
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

	if err := apiFor(r.clients, r.product, projectID).Delete(state.ID.ValueString()); err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting database", common.FormatError(err))
	}
}

func (r *databaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *databaseResource) mapToState(database *models.Database, model *databaseResourceModel) {
	model.ID = types.StringValue(database.Id)
	model.Name = types.StringValue(database.Name)
	model.Enabled = types.BoolValue(database.Enabled)
	model.Specification = types.StringValue(database.Specification)
	model.Replicas = types.Int64Value(int64(database.Replicas))
	model.Type = types.StringValue(database.Type)
	model.Status = types.StringValue(database.Status)
	model.Engine = types.StringValue(database.Engine)
	model.CreatedAt = types.StringValue(database.CreatedAt)
	model.UpdatedAt = types.StringValue(database.UpdatedAt)

	// The database model carries no sync mode, so a configured value is kept
	// rather than blanked out on refresh.
	if model.SyncMode.IsUnknown() {
		model.SyncMode = types.StringNull()
	}
}
