package database

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v2/id"
	"github.com/appwrite/sdk-for-go/v2/tablesdb"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &databaseResource{}
	_ resource.ResourceWithConfigure   = &databaseResource{}
	_ resource.ResourceWithImportState = &databaseResource{}
)

type databaseResource struct {
	tablesdb *tablesdb.TablesDB
}

type databaseResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Enabled   types.Bool   `tfsdk:"enabled"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewDatabaseResource() resource.Resource {
	return &databaseResource{}
}

func (r *databaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tablesdb"
}

func (r *databaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite database.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The database ID. Must be unique within the project.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The database name.",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the database is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"created_at": schema.StringAttribute{
				Description: "The database creation timestamp.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The database last update timestamp.",
				Computed:    true,
			},
		},
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
	r.tablesdb = clients.TablesDB
}

func (r *databaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan databaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dbID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		dbID = id.Unique()
	}

	var opts []tablesdb.CreateOption
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		opts = append(opts, r.tablesdb.WithCreateEnabled(plan.Enabled.ValueBool()))
	}

	db, err := r.tablesdb.Create(dbID, plan.Name.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error creating database", common.FormatError(err))
		return
	}

	plan.ID = types.StringValue(db.Id)
	plan.Name = types.StringValue(db.Name)
	plan.Enabled = types.BoolValue(db.Enabled)
	plan.CreatedAt = types.StringValue(db.CreatedAt)
	plan.UpdatedAt = types.StringValue(db.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *databaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state databaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	db, err := r.tablesdb.Get(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading database", common.FormatError(err))
		return
	}

	state.ID = types.StringValue(db.Id)
	state.Name = types.StringValue(db.Name)
	state.Enabled = types.BoolValue(db.Enabled)
	state.CreatedAt = types.StringValue(db.CreatedAt)
	state.UpdatedAt = types.StringValue(db.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *databaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan databaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := []tablesdb.UpdateOption{
		r.tablesdb.WithUpdateName(plan.Name.ValueString()),
		r.tablesdb.WithUpdateEnabled(plan.Enabled.ValueBool()),
	}

	db, err := r.tablesdb.Update(plan.ID.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error updating database", common.FormatError(err))
		return
	}

	plan.ID = types.StringValue(db.Id)
	plan.Name = types.StringValue(db.Name)
	plan.Enabled = types.BoolValue(db.Enabled)
	plan.CreatedAt = types.StringValue(db.CreatedAt)
	plan.UpdatedAt = types.StringValue(db.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *databaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state databaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.tablesdb.Delete(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting database", common.FormatError(err))
	}
}

func (r *databaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
