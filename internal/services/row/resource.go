package row

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v2/id"
	"github.com/appwrite/sdk-for-go/v2/models"
	"github.com/appwrite/sdk-for-go/v2/tablesdb"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &rowResource{}
	_ resource.ResourceWithConfigure   = &rowResource{}
	_ resource.ResourceWithImportState = &rowResource{}
)

type rowResource struct {
	tablesdb  *tablesdb.TablesDB
	projectID string
}

type rowResourceModel struct {
	ID          types.String `tfsdk:"id"`
	DatabaseID  types.String `tfsdk:"database_id"`
	TableID     types.String `tfsdk:"table_id"`
	Data        types.String `tfsdk:"data"`
	Permissions types.List   `tfsdk:"permissions"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	ProjectID   types.String `tfsdk:"project_id"`
}

func NewRowResource() resource.Resource {
	return &rowResource{}
}

func (r *rowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tablesdb_row"
}

func (r *rowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a row in an Appwrite tablesdb table.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The row ID.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
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
			"data": schema.StringAttribute{
				Description: "The row data as a JSON object. Keys are column keys, values are column values.",
				Required:    true,
			},
			"permissions": schema.ListAttribute{
				Description: "Row permissions.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.StringAttribute{
				Description: "The row creation timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The row last update timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

func (r *rowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData))
		return
	}
	r.tablesdb = clients.TablesDB
	r.projectID = clients.ProjectID
}

func (r *rowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan rowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(plan.Data.ValueString()), &data); err != nil {
		resp.Diagnostics.AddError("Invalid data JSON", fmt.Sprintf("Failed to parse data: %s", err))
		return
	}

	rowID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		rowID = id.Unique()
	}

	var opts []tablesdb.CreateRowOption
	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var perms []string
		resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &perms, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, r.tablesdb.WithCreateRowPermissions(perms))
	}

	row, err := r.tablesdb.CreateRow(
		plan.DatabaseID.ValueString(),
		plan.TableID.ValueString(),
		rowID,
		data,
		opts...,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating row", common.FormatError(err))
		return
	}

	r.mapToState(ctx, row, &plan, &resp.Diagnostics)
	if plan.ProjectID.IsNull() || plan.ProjectID.IsUnknown() {
		plan.ProjectID = types.StringValue(r.projectID)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *rowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state rowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	row, err := r.tablesdb.GetRow(
		state.DatabaseID.ValueString(),
		state.TableID.ValueString(),
		state.ID.ValueString(),
	)
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading row", common.FormatError(err))
		return
	}

	r.mapToState(ctx, row, &state, &resp.Diagnostics)
	if state.ProjectID.IsNull() || state.ProjectID.IsUnknown() {
		state.ProjectID = types.StringValue(r.projectID)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *rowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan rowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var opts []tablesdb.UpdateRowOption

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(plan.Data.ValueString()), &data); err != nil {
		resp.Diagnostics.AddError("Invalid data JSON", fmt.Sprintf("Failed to parse data: %s", err))
		return
	}
	opts = append(opts, r.tablesdb.WithUpdateRowData(data))

	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var perms []string
		resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &perms, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, r.tablesdb.WithUpdateRowPermissions(perms))
	}

	row, err := r.tablesdb.UpdateRow(
		plan.DatabaseID.ValueString(),
		plan.TableID.ValueString(),
		plan.ID.ValueString(),
		opts...,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error updating row", common.FormatError(err))
		return
	}

	r.mapToState(ctx, row, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *rowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state rowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.tablesdb.DeleteRow(
		state.DatabaseID.ValueString(),
		state.TableID.ValueString(),
		state.ID.ValueString(),
	)
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting row", common.FormatError(err))
	}
}

func (r *rowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: database_id/table_id/row_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: database_id/table_id/row_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("table_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

func (r *rowResource) mapToState(ctx context.Context, row *models.Row, model *rowResourceModel, diagnostics *diag.Diagnostics) {
	model.ID = types.StringValue(row.Id)
	model.DatabaseID = types.StringValue(row.DatabaseId)
	model.TableID = types.StringValue(row.TableId)
	model.CreatedAt = types.StringValue(row.CreatedAt)
	model.UpdatedAt = types.StringValue(row.UpdatedAt)

	permsList, diags := types.ListValueFrom(ctx, types.StringType, row.Permissions)
	diagnostics.Append(diags...)
	model.Permissions = permsList

	// Decode the full row response to extract user data (excluding system fields)
	var rawData map[string]interface{}
	if err := row.Decode(&rawData); err != nil {
		diagnostics.AddError("Error decoding row data", err.Error())
		return
	}

	// Parse the plan data to get the keys the user specified
	var planData map[string]interface{}
	if !model.Data.IsNull() && !model.Data.IsUnknown() {
		if err := json.Unmarshal([]byte(model.Data.ValueString()), &planData); err != nil {
			planData = nil
		}
	}

	// Filter response to only include keys from the plan (or all non-system keys on import)
	filtered := make(map[string]interface{})
	for key, val := range rawData {
		if strings.HasPrefix(key, "$") {
			continue
		}
		if planData == nil {
			filtered[key] = val
		} else if _, exists := planData[key]; exists {
			filtered[key] = val
		}
	}

	dataJSON, err := json.Marshal(filtered)
	if err != nil {
		diagnostics.AddError("Error encoding row data", err.Error())
		return
	}
	model.Data = types.StringValue(string(dataJSON))
}
