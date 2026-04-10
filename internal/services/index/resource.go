package index

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/appwrite/sdk-for-go/v2/id"
	"github.com/appwrite/sdk-for-go/v2/tablesdb"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &indexResource{}
	_ resource.ResourceWithConfigure   = &indexResource{}
	_ resource.ResourceWithImportState = &indexResource{}
)

type indexResource struct {
	tablesdb *tablesdb.TablesDB
}

type indexResourceModel struct {
	DatabaseID types.String `tfsdk:"database_id"`
	TableID    types.String `tfsdk:"table_id"`
	Key        types.String `tfsdk:"key"`
	Type       types.String `tfsdk:"type"`
	Columns    types.List   `tfsdk:"columns"`
	Orders     types.List   `tfsdk:"orders"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

func NewIndexResource() resource.Resource {
	return &indexResource{}
}

func (r *indexResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tablesdb_index"
}

func (r *indexResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an index on an Appwrite table.",
		Attributes: map[string]schema.Attribute{
			"database_id": schema.StringAttribute{
				Description:   "The database ID.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"table_id": schema.StringAttribute{
				Description:   "The table ID.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key": schema.StringAttribute{
				Description:   "The index key (name).",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				Description:   "Index type: key, unique, or fulltext.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"columns": schema.ListAttribute{
				Description:   "Array of column keys to index.",
				Required:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{},
			},
			"orders": schema.ListAttribute{
				Description: "Array of index orders (asc or desc) for each column.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *indexResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData))
		return
	}
	r.tablesdb = clients.TablesDB
}

func (r *indexResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan indexResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var columns []string
	resp.Diagnostics.Append(plan.Columns.ElementsAs(ctx, &columns, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var opts []tablesdb.CreateIndexOption
	if !plan.Orders.IsNull() && !plan.Orders.IsUnknown() {
		var orders []string
		resp.Diagnostics.Append(plan.Orders.ElementsAs(ctx, &orders, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, r.tablesdb.WithCreateIndexOrders(orders))
	}

	// Appwrite creates columns asynchronously. Wait for all columns to become available.
	databaseId := plan.DatabaseID.ValueString()
	tableId := plan.TableID.ValueString()
	if err := r.waitForColumns(ctx, databaseId, tableId, columns); err != nil {
		resp.Diagnostics.AddError("Error waiting for columns", err.Error())
		return
	}

	indexKey := plan.Key.ValueString()
	if plan.Key.IsNull() || plan.Key.IsUnknown() {
		indexKey = id.Unique()
	}

	idx, err := r.tablesdb.CreateIndex(
		databaseId,
		tableId,
		indexKey,
		plan.Type.ValueString(),
		columns,
		opts...,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating index", common.FormatError(err))
		return
	}

	plan.Key = types.StringValue(idx.Key)
	plan.Type = types.StringValue(idx.Type)
	plan.CreatedAt = types.StringValue(idx.CreatedAt)
	plan.UpdatedAt = types.StringValue(idx.UpdatedAt)
	colList, diags := types.ListValueFrom(ctx, types.StringType, idx.Columns)
	resp.Diagnostics.Append(diags...)
	plan.Columns = colList
	if len(idx.Orders) > 0 {
		orderList, diags := types.ListValueFrom(ctx, types.StringType, idx.Orders)
		resp.Diagnostics.Append(diags...)
		plan.Orders = orderList
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *indexResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state indexResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idx, err := r.tablesdb.GetIndex(state.DatabaseID.ValueString(), state.TableID.ValueString(), state.Key.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading index", common.FormatError(err))
		return
	}

	state.Key = types.StringValue(idx.Key)
	state.Type = types.StringValue(idx.Type)
	state.CreatedAt = types.StringValue(idx.CreatedAt)
	state.UpdatedAt = types.StringValue(idx.UpdatedAt)
	colList, diags := types.ListValueFrom(ctx, types.StringType, idx.Columns)
	resp.Diagnostics.Append(diags...)
	state.Columns = colList
	if len(idx.Orders) > 0 {
		orderList, diags := types.ListValueFrom(ctx, types.StringType, idx.Orders)
		resp.Diagnostics.Append(diags...)
		state.Orders = orderList
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is not supported for indexes - they must be recreated.
func (r *indexResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Indexes cannot be updated in place. Delete and recreate instead.")
}

func (r *indexResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state indexResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.tablesdb.DeleteIndex(state.DatabaseID.ValueString(), state.TableID.ValueString(), state.Key.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting index", common.FormatError(err))
	}
}

func (r *indexResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: database_id/table_id/key, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("table_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), parts[2])...)
}

// waitForColumns polls each column until its status is "available" or the context is cancelled.
func (r *indexResource) waitForColumns(ctx context.Context, databaseId, tableId string, columns []string) error {
	for _, col := range columns {
		for {
			select {
			case <-ctx.Done():
				return fmt.Errorf("timed out waiting for column %q to become available", col)
			default:
			}

			raw, err := r.tablesdb.GetColumn(databaseId, tableId, col)
			if err != nil {
				return fmt.Errorf("error checking column %q status: %w", col, err)
			}

			status, _ := common.GetColumnStatus(raw)
			if status == "available" {
				break
			}
			if status == "failed" || status == "stuck" {
				return fmt.Errorf("column %q is in %q state and cannot be indexed", col, status)
			}

			time.Sleep(1 * time.Second)
		}
	}
	return nil
}
