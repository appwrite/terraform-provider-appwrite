package table

import (
	"context"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v2/tablesdb"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &tableResource{}
	_ resource.ResourceWithConfigure   = &tableResource{}
	_ resource.ResourceWithImportState = &tableResource{}
)

type tableResource struct {
	tablesdb *tablesdb.TablesDB
}

type tableResourceModel struct {
	ID          types.String `tfsdk:"id"`
	DatabaseID  types.String `tfsdk:"database_id"`
	Name        types.String `tfsdk:"name"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	RowSecurity types.Bool   `tfsdk:"row_security"`
	Permissions types.List   `tfsdk:"permissions"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func NewTableResource() resource.Resource {
	return &tableResource{}
}

func (r *tableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_table"
}

func (r *tableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite table within a database.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The table ID. Must be unique within the database.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"database_id": schema.StringAttribute{
				Description: "The ID of the database this table belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The table name.",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the table is enabled. When disabled, the table is inaccessible to users but remains accessible via API keys. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"row_security": schema.BoolAttribute{
				Description: "Whether row-level permissions are enabled. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"permissions": schema.ListAttribute{
				Description: "Table-level permissions.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The table creation timestamp.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The table last update timestamp.",
				Computed:    true,
			},
		},
	}
}

func (r *tableResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *tableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var opts []tablesdb.CreateTableOption
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		opts = append(opts, r.tablesdb.WithCreateTableEnabled(plan.Enabled.ValueBool()))
	}
	if !plan.RowSecurity.IsNull() && !plan.RowSecurity.IsUnknown() {
		opts = append(opts, r.tablesdb.WithCreateTableRowSecurity(plan.RowSecurity.ValueBool()))
	}
	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var perms []string
		resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &perms, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, r.tablesdb.WithCreateTablePermissions(perms))
	}

	table, err := r.tablesdb.CreateTable(
		plan.DatabaseID.ValueString(),
		plan.ID.ValueString(),
		plan.Name.ValueString(),
		opts...,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating table", common.FormatError(err))
		return
	}

	plan.ID = types.StringValue(table.Id)
	plan.DatabaseID = types.StringValue(table.DatabaseId)
	plan.Name = types.StringValue(table.Name)
	plan.Enabled = types.BoolValue(table.Enabled)
	plan.RowSecurity = types.BoolValue(table.RowSecurity)
	plan.CreatedAt = types.StringValue(table.CreatedAt)
	plan.UpdatedAt = types.StringValue(table.UpdatedAt)
	permsList, diags := types.ListValueFrom(ctx, types.StringType, table.Permissions)
	resp.Diagnostics.Append(diags...)
	plan.Permissions = permsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	table, err := r.tablesdb.GetTable(state.DatabaseID.ValueString(), state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading table", common.FormatError(err))
		return
	}

	state.ID = types.StringValue(table.Id)
	state.DatabaseID = types.StringValue(table.DatabaseId)
	state.Name = types.StringValue(table.Name)
	state.Enabled = types.BoolValue(table.Enabled)
	state.RowSecurity = types.BoolValue(table.RowSecurity)
	state.CreatedAt = types.StringValue(table.CreatedAt)
	state.UpdatedAt = types.StringValue(table.UpdatedAt)
	permsList, diags := types.ListValueFrom(ctx, types.StringType, table.Permissions)
	resp.Diagnostics.Append(diags...)
	state.Permissions = permsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *tableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan tableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := []tablesdb.UpdateTableOption{
		r.tablesdb.WithUpdateTableName(plan.Name.ValueString()),
		r.tablesdb.WithUpdateTableEnabled(plan.Enabled.ValueBool()),
		r.tablesdb.WithUpdateTableRowSecurity(plan.RowSecurity.ValueBool()),
	}

	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var perms []string
		resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &perms, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, r.tablesdb.WithUpdateTablePermissions(perms))
	}

	table, err := r.tablesdb.UpdateTable(plan.DatabaseID.ValueString(), plan.ID.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error updating table", common.FormatError(err))
		return
	}

	plan.ID = types.StringValue(table.Id)
	plan.DatabaseID = types.StringValue(table.DatabaseId)
	plan.Name = types.StringValue(table.Name)
	plan.Enabled = types.BoolValue(table.Enabled)
	plan.RowSecurity = types.BoolValue(table.RowSecurity)
	plan.CreatedAt = types.StringValue(table.CreatedAt)
	plan.UpdatedAt = types.StringValue(table.UpdatedAt)
	permsList, diags := types.ListValueFrom(ctx, types.StringType, table.Permissions)
	resp.Diagnostics.Append(diags...)
	plan.Permissions = permsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state tableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.tablesdb.DeleteTable(state.DatabaseID.ValueString(), state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting table", common.FormatError(err))
	}
}

// ImportState supports importing via "database_id/table_id" format.
func (r *tableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: database_id/table_id, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
