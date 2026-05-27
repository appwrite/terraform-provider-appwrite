package table

import (
	"context"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v4/appwrite"
	"github.com/appwrite/sdk-for-go/v4/id"
	"github.com/appwrite/sdk-for-go/v4/tablesdb"
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
	clients *common.AppwriteClients
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
	ProjectID   types.String `tfsdk:"project_id"`
}

func NewTableResource() resource.Resource {
	return &tableResource{}
}

func (r *tableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tablesdb_table"
}

func (r *tableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite table within a database.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The table ID. Must be unique within the database.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"database_id": schema.StringAttribute{
				Description:   "The ID of the database this table belongs to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
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
				Description:   "Table-level permissions.",
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Description: "The table creation timestamp.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The table last update timestamp.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					common.UseStateForUnknownUnlessUpdating(),
				},
			},
			"project_id": common.ProjectIDAttribute(),
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
	r.clients = clients
}

func (r *tableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	tablesdbClient := appwrite.NewTablesDB(r.clients.ClientForProject(projectID))

	var opts []tablesdb.CreateTableOption
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		opts = append(opts, tablesdbClient.WithCreateTableEnabled(plan.Enabled.ValueBool()))
	}
	if !plan.RowSecurity.IsNull() && !plan.RowSecurity.IsUnknown() {
		opts = append(opts, tablesdbClient.WithCreateTableRowSecurity(plan.RowSecurity.ValueBool()))
	}
	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var perms []string
		resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &perms, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, tablesdbClient.WithCreateTablePermissions(perms))
	}

	tableID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		tableID = id.Unique()
	}

	table, err := tablesdbClient.CreateTable(
		plan.DatabaseID.ValueString(),
		tableID,
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
	plan.ProjectID = types.StringValue(projectID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	tablesdbClient := appwrite.NewTablesDB(r.clients.ClientForProject(projectID))

	table, err := tablesdbClient.GetTable(state.DatabaseID.ValueString(), state.ID.ValueString())
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
	state.ProjectID = types.StringValue(projectID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *tableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan tableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	tablesdbClient := appwrite.NewTablesDB(r.clients.ClientForProject(projectID))

	opts := []tablesdb.UpdateTableOption{
		tablesdbClient.WithUpdateTableName(plan.Name.ValueString()),
		tablesdbClient.WithUpdateTableEnabled(plan.Enabled.ValueBool()),
		tablesdbClient.WithUpdateTableRowSecurity(plan.RowSecurity.ValueBool()),
	}

	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var perms []string
		resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &perms, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, tablesdbClient.WithUpdateTablePermissions(perms))
	}

	table, err := tablesdbClient.UpdateTable(plan.DatabaseID.ValueString(), plan.ID.ValueString(), opts...)
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

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	tablesdbClient := appwrite.NewTablesDB(r.clients.ClientForProject(projectID))

	_, err = tablesdbClient.DeleteTable(state.DatabaseID.ValueString(), state.ID.ValueString())
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
