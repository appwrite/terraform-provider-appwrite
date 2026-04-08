package column

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/appwrite/sdk-for-go/v2/tablesdb"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &columnResource{}
	_ resource.ResourceWithConfigure   = &columnResource{}
	_ resource.ResourceWithImportState = &columnResource{}
)

var allColumnTypes = "varchar, text, longtext, mediumtext, integer, float, boolean, enum, email, datetime, url, ip, point, line, polygon, relationship, string"

type columnResource struct {
	tablesdb *tablesdb.TablesDB
}

type columnResourceModel struct {
	DatabaseID     types.String  `tfsdk:"database_id"`
	TableID        types.String  `tfsdk:"table_id"`
	Key            types.String  `tfsdk:"key"`
	Type           types.String  `tfsdk:"type"`
	Required       types.Bool    `tfsdk:"required"`
	Array          types.Bool    `tfsdk:"array"`
	Size           types.Int64   `tfsdk:"size"`
	Min            types.Int64   `tfsdk:"min"`
	Max            types.Int64   `tfsdk:"max"`
	FloatMin       types.Float64 `tfsdk:"float_min"`
	FloatMax       types.Float64 `tfsdk:"float_max"`
	Elements       types.List    `tfsdk:"elements"`
	DefaultStr     types.String  `tfsdk:"default"`
	Encrypt        types.Bool    `tfsdk:"encrypt"`
	RelatedTableID types.String  `tfsdk:"related_table_id"`
	RelationType   types.String  `tfsdk:"relationship_type"`
	TwoWay         types.Bool    `tfsdk:"two_way"`
	TwoWayKey      types.String  `tfsdk:"two_way_key"`
	OnDelete       types.String  `tfsdk:"on_delete"`
	CreatedAt      types.String  `tfsdk:"created_at"`
	UpdatedAt      types.String  `tfsdk:"updated_at"`
}

func NewColumnResource() resource.Resource {
	return &columnResource{}
}

func (r *columnResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_column"
}

func (r *columnResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a column in an Appwrite table.",
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
				Description:   "The column key (name).",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				Description:   "The column type. One of: " + allColumnTypes + ".",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"required": schema.BoolAttribute{
				Description: "Whether the column is required.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"array": schema.BoolAttribute{
				Description: "Whether the column is an array. Applies to string, varchar, text, longtext, mediumtext, integer, float, boolean, enum, email, datetime, url, ip types.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"size": schema.Int64Attribute{
				Description: "Maximum length. Required for string and varchar types.",
				Optional:    true,
			},
			"min": schema.Int64Attribute{
				Description: "Minimum value. Applies to integer type.",
				Optional:    true,
			},
			"max": schema.Int64Attribute{
				Description: "Maximum value. Applies to integer type.",
				Optional:    true,
			},
			"float_min": schema.Float64Attribute{
				Description: "Minimum value. Applies to float type.",
				Optional:    true,
			},
			"float_max": schema.Float64Attribute{
				Description: "Maximum value. Applies to float type.",
				Optional:    true,
			},
			"elements": schema.ListAttribute{
				Description: "Array of allowed values. Required for enum type.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"default": schema.StringAttribute{
				Description: "Default value. Use \"true\"/\"false\" for boolean, numeric strings for integer/float.",
				Optional:    true,
			},
			"encrypt": schema.BoolAttribute{
				Description: "Whether the column is encrypted. Applies to string, varchar, text, longtext, mediumtext types.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"related_table_id": schema.StringAttribute{
				Description:   "The ID of the related table. Required for relationship type.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"relationship_type": schema.StringAttribute{
				Description:   "The relationship type: oneToOne, oneToMany, manyToOne, manyToMany. Required for relationship type.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"two_way": schema.BoolAttribute{
				Description: "Whether the relationship is two-way. Applies to relationship type.",
				Optional:    true,
			},
			"two_way_key": schema.StringAttribute{
				Description: "The key for the two-way relationship column. Applies to relationship type.",
				Optional:    true,
			},
			"on_delete": schema.StringAttribute{
				Description: "Behavior when the parent document is deleted: restrict, cascade, setNull. Applies to relationship type.",
				Optional:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The column creation timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The column last update timestamp in ISO 8601 format.",
				Computed:    true,
			},
		},
	}
}

func (r *columnResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *columnResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan columnResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	databaseId := plan.DatabaseID.ValueString()
	tableId := plan.TableID.ValueString()
	key := plan.Key.ValueString()
	required := plan.Required.ValueBool()
	columnType := plan.Type.ValueString()
	array := plan.Array.ValueBool()

	var responseJSON []byte
	var err error

	switch columnType {
	case "string":
		if plan.Size.IsNull() {
			resp.Diagnostics.AddError("Missing attribute", "size is required for string columns")
			return
		}
		var opts []tablesdb.CreateStringColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateStringColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, r.tablesdb.WithCreateStringColumnArray(array))
		if !plan.Encrypt.IsNull() && !plan.Encrypt.IsUnknown() {
			opts = append(opts, r.tablesdb.WithCreateStringColumnEncrypt(plan.Encrypt.ValueBool()))
		}
		col, e := r.tablesdb.CreateStringColumn(databaseId, tableId, key, int(plan.Size.ValueInt64()), required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "varchar":
		if plan.Size.IsNull() {
			resp.Diagnostics.AddError("Missing attribute", "size is required for varchar columns")
			return
		}
		var opts []tablesdb.CreateVarcharColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateVarcharColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, r.tablesdb.WithCreateVarcharColumnArray(array))
		if !plan.Encrypt.IsNull() && !plan.Encrypt.IsUnknown() {
			opts = append(opts, r.tablesdb.WithCreateVarcharColumnEncrypt(plan.Encrypt.ValueBool()))
		}
		col, e := r.tablesdb.CreateVarcharColumn(databaseId, tableId, key, int(plan.Size.ValueInt64()), required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "text":
		var opts []tablesdb.CreateTextColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateTextColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, r.tablesdb.WithCreateTextColumnArray(array))
		if !plan.Encrypt.IsNull() && !plan.Encrypt.IsUnknown() {
			opts = append(opts, r.tablesdb.WithCreateTextColumnEncrypt(plan.Encrypt.ValueBool()))
		}
		col, e := r.tablesdb.CreateTextColumn(databaseId, tableId, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "longtext":
		var opts []tablesdb.CreateLongtextColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateLongtextColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, r.tablesdb.WithCreateLongtextColumnArray(array))
		if !plan.Encrypt.IsNull() && !plan.Encrypt.IsUnknown() {
			opts = append(opts, r.tablesdb.WithCreateLongtextColumnEncrypt(plan.Encrypt.ValueBool()))
		}
		col, e := r.tablesdb.CreateLongtextColumn(databaseId, tableId, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "mediumtext":
		var opts []tablesdb.CreateMediumtextColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateMediumtextColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, r.tablesdb.WithCreateMediumtextColumnArray(array))
		if !plan.Encrypt.IsNull() && !plan.Encrypt.IsUnknown() {
			opts = append(opts, r.tablesdb.WithCreateMediumtextColumnEncrypt(plan.Encrypt.ValueBool()))
		}
		col, e := r.tablesdb.CreateMediumtextColumn(databaseId, tableId, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "integer":
		var opts []tablesdb.CreateIntegerColumnOption
		if !plan.Min.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateIntegerColumnMin(int(plan.Min.ValueInt64())))
		}
		if !plan.Max.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateIntegerColumnMax(int(plan.Max.ValueInt64())))
		}
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateIntegerColumnDefault(parseIntDefault(plan.DefaultStr.ValueString())))
		}
		opts = append(opts, r.tablesdb.WithCreateIntegerColumnArray(array))
		col, e := r.tablesdb.CreateIntegerColumn(databaseId, tableId, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "float":
		var opts []tablesdb.CreateFloatColumnOption
		if !plan.FloatMin.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateFloatColumnMin(plan.FloatMin.ValueFloat64()))
		}
		if !plan.FloatMax.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateFloatColumnMax(plan.FloatMax.ValueFloat64()))
		}
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateFloatColumnDefault(parseFloatDefault(plan.DefaultStr.ValueString())))
		}
		opts = append(opts, r.tablesdb.WithCreateFloatColumnArray(array))
		col, e := r.tablesdb.CreateFloatColumn(databaseId, tableId, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "boolean":
		var opts []tablesdb.CreateBooleanColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateBooleanColumnDefault(plan.DefaultStr.ValueString() == "true"))
		}
		opts = append(opts, r.tablesdb.WithCreateBooleanColumnArray(array))
		col, e := r.tablesdb.CreateBooleanColumn(databaseId, tableId, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "enum":
		if plan.Elements.IsNull() {
			resp.Diagnostics.AddError("Missing attribute", "elements is required for enum columns")
			return
		}
		var elements []string
		resp.Diagnostics.Append(plan.Elements.ElementsAs(ctx, &elements, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		var opts []tablesdb.CreateEnumColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateEnumColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, r.tablesdb.WithCreateEnumColumnArray(array))
		col, e := r.tablesdb.CreateEnumColumn(databaseId, tableId, key, elements, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "email":
		var opts []tablesdb.CreateEmailColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateEmailColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, r.tablesdb.WithCreateEmailColumnArray(array))
		col, e := r.tablesdb.CreateEmailColumn(databaseId, tableId, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "datetime":
		var opts []tablesdb.CreateDatetimeColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateDatetimeColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, r.tablesdb.WithCreateDatetimeColumnArray(array))
		col, e := r.tablesdb.CreateDatetimeColumn(databaseId, tableId, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "url":
		var opts []tablesdb.CreateUrlColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateUrlColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, r.tablesdb.WithCreateUrlColumnArray(array))
		col, e := r.tablesdb.CreateUrlColumn(databaseId, tableId, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "ip":
		var opts []tablesdb.CreateIpColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateIpColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, r.tablesdb.WithCreateIpColumnArray(array))
		col, e := r.tablesdb.CreateIpColumn(databaseId, tableId, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "point":
		col, e := r.tablesdb.CreatePointColumn(databaseId, tableId, key, required)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "line":
		col, e := r.tablesdb.CreateLineColumn(databaseId, tableId, key, required)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "polygon":
		col, e := r.tablesdb.CreatePolygonColumn(databaseId, tableId, key, required)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "relationship":
		if plan.RelatedTableID.IsNull() {
			resp.Diagnostics.AddError("Missing attribute", "related_table_id is required for relationship columns")
			return
		}
		if plan.RelationType.IsNull() {
			resp.Diagnostics.AddError("Missing attribute", "relationship_type is required for relationship columns")
			return
		}
		var opts []tablesdb.CreateRelationshipColumnOption
		if !plan.TwoWay.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateRelationshipColumnTwoWay(plan.TwoWay.ValueBool()))
		}
		if !plan.Key.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateRelationshipColumnKey(key))
		}
		if !plan.TwoWayKey.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateRelationshipColumnTwoWayKey(plan.TwoWayKey.ValueString()))
		}
		if !plan.OnDelete.IsNull() {
			opts = append(opts, r.tablesdb.WithCreateRelationshipColumnOnDelete(plan.OnDelete.ValueString()))
		}
		col, e := r.tablesdb.CreateRelationshipColumn(databaseId, tableId, plan.RelatedTableID.ValueString(), plan.RelationType.ValueString(), opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	default:
		resp.Diagnostics.AddError("Unsupported column type", fmt.Sprintf("Column type %q is not supported. Use one of: %s.", columnType, allColumnTypes))
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error creating column", common.FormatError(err))
		return
	}

	r.readResponseIntoState(ctx, responseJSON, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *columnResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state columnResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, err := r.tablesdb.GetColumn(state.DatabaseID.ValueString(), state.TableID.ValueString(), state.Key.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading column", common.FormatError(err))
		return
	}

	var generic map[string]interface{}
	if err := common.DecodeColumn(raw, &generic); err != nil {
		resp.Diagnostics.AddError("Error decoding column", err.Error())
		return
	}

	responseJSON, _ := json.Marshal(generic)
	r.readResponseIntoState(ctx, responseJSON, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *columnResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan columnResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	databaseId := plan.DatabaseID.ValueString()
	tableId := plan.TableID.ValueString()
	key := plan.Key.ValueString()
	required := plan.Required.ValueBool()
	columnType := plan.Type.ValueString()

	defaultStr := ""
	if !plan.DefaultStr.IsNull() {
		defaultStr = plan.DefaultStr.ValueString()
	}

	var responseJSON []byte
	var err error

	switch columnType {
	case "string":
		var opts []tablesdb.UpdateStringColumnOption
		if !plan.Size.IsNull() {
			opts = append(opts, r.tablesdb.WithUpdateStringColumnSize(int(plan.Size.ValueInt64())))
		}
		col, e := r.tablesdb.UpdateStringColumn(databaseId, tableId, key, required, defaultStr, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "varchar":
		var opts []tablesdb.UpdateVarcharColumnOption
		if !plan.Size.IsNull() {
			opts = append(opts, r.tablesdb.WithUpdateVarcharColumnSize(int(plan.Size.ValueInt64())))
		}
		col, e := r.tablesdb.UpdateVarcharColumn(databaseId, tableId, key, required, defaultStr, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "text":
		col, e := r.tablesdb.UpdateTextColumn(databaseId, tableId, key, required, defaultStr)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "longtext":
		col, e := r.tablesdb.UpdateLongtextColumn(databaseId, tableId, key, required, defaultStr)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "mediumtext":
		col, e := r.tablesdb.UpdateMediumtextColumn(databaseId, tableId, key, required, defaultStr)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "integer":
		var opts []tablesdb.UpdateIntegerColumnOption
		if !plan.Min.IsNull() {
			opts = append(opts, r.tablesdb.WithUpdateIntegerColumnMin(int(plan.Min.ValueInt64())))
		}
		if !plan.Max.IsNull() {
			opts = append(opts, r.tablesdb.WithUpdateIntegerColumnMax(int(plan.Max.ValueInt64())))
		}
		col, e := r.tablesdb.UpdateIntegerColumn(databaseId, tableId, key, required, parseIntDefault(defaultStr), opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "float":
		var opts []tablesdb.UpdateFloatColumnOption
		if !plan.FloatMin.IsNull() {
			opts = append(opts, r.tablesdb.WithUpdateFloatColumnMin(plan.FloatMin.ValueFloat64()))
		}
		if !plan.FloatMax.IsNull() {
			opts = append(opts, r.tablesdb.WithUpdateFloatColumnMax(plan.FloatMax.ValueFloat64()))
		}
		col, e := r.tablesdb.UpdateFloatColumn(databaseId, tableId, key, required, parseFloatDefault(defaultStr), opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "boolean":
		col, e := r.tablesdb.UpdateBooleanColumn(databaseId, tableId, key, required, defaultStr == "true")
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "enum":
		var elements []string
		resp.Diagnostics.Append(plan.Elements.ElementsAs(ctx, &elements, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		col, e := r.tablesdb.UpdateEnumColumn(databaseId, tableId, key, elements, required, defaultStr)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "email":
		col, e := r.tablesdb.UpdateEmailColumn(databaseId, tableId, key, required, defaultStr)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "datetime":
		col, e := r.tablesdb.UpdateDatetimeColumn(databaseId, tableId, key, required, defaultStr)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "url":
		col, e := r.tablesdb.UpdateUrlColumn(databaseId, tableId, key, required, defaultStr)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "ip":
		col, e := r.tablesdb.UpdateIpColumn(databaseId, tableId, key, required, defaultStr)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "point":
		col, e := r.tablesdb.UpdatePointColumn(databaseId, tableId, key, required)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "line":
		col, e := r.tablesdb.UpdateLineColumn(databaseId, tableId, key, required)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "polygon":
		col, e := r.tablesdb.UpdatePolygonColumn(databaseId, tableId, key, required)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "relationship":
		var opts []tablesdb.UpdateRelationshipColumnOption
		if !plan.OnDelete.IsNull() {
			opts = append(opts, r.tablesdb.WithUpdateRelationshipColumnOnDelete(plan.OnDelete.ValueString()))
		}
		col, e := r.tablesdb.UpdateRelationshipColumn(databaseId, tableId, key, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}
	}

	if err != nil {
		resp.Diagnostics.AddError("Error updating column", common.FormatError(err))
		return
	}

	r.readResponseIntoState(ctx, responseJSON, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *columnResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state columnResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.tablesdb.DeleteColumn(state.DatabaseID.ValueString(), state.TableID.ValueString(), state.Key.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting column", common.FormatError(err))
	}
}

func (r *columnResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	common.ImportColumnState(ctx, req, resp)
}

// readResponseIntoState maps a generic JSON column response into the Terraform state model.
func (r *columnResource) readResponseIntoState(ctx context.Context, responseJSON []byte, model *columnResourceModel, diagnostics *diag.Diagnostics) {
	var generic map[string]interface{}
	if err := json.Unmarshal(responseJSON, &generic); err != nil {
		diagnostics.AddError("Error parsing column response", err.Error())
		return
	}

	if key, ok := generic["key"].(string); ok {
		model.Key = types.StringValue(key)
	}
	// Don't overwrite type from API — Appwrite returns internal type names
	// (e.g. "double" for "float", "string" for "email"/"enum") which differ
	// from the user-facing type names we use in the schema.
	if required, ok := generic["required"].(bool); ok {
		model.Required = types.BoolValue(required)
	}
	if array, ok := generic["array"].(bool); ok {
		model.Array = types.BoolValue(array)
	}
	if createdAt, ok := generic["$createdAt"].(string); ok {
		model.CreatedAt = types.StringValue(createdAt)
	}
	if updatedAt, ok := generic["$updatedAt"].(string); ok {
		model.UpdatedAt = types.StringValue(updatedAt)
	}
	if size, ok := generic["size"].(float64); ok && size > 0 {
		model.Size = types.Int64Value(int64(size))
	}
	if min, ok := generic["min"].(float64); ok {
		columnType := model.Type.ValueString()
		if columnType == "integer" {
			model.Min = types.Int64Value(int64(min))
		} else if columnType == "float" {
			model.FloatMin = types.Float64Value(min)
		}
	}
	if max, ok := generic["max"].(float64); ok {
		columnType := model.Type.ValueString()
		if columnType == "integer" {
			model.Max = types.Int64Value(int64(max))
		} else if columnType == "float" {
			model.FloatMax = types.Float64Value(max)
		}
	}
	if encrypt, ok := generic["encrypt"].(bool); ok {
		model.Encrypt = types.BoolValue(encrypt)
	}
	if elements, ok := generic["elements"].([]interface{}); ok && len(elements) > 0 {
		strs := make([]string, len(elements))
		for i, e := range elements {
			strs[i], _ = e.(string)
		}
		elemList, diags := types.ListValueFrom(ctx, types.StringType, strs)
		diagnostics.Append(diags...)
		model.Elements = elemList
	}

	// Relationship-specific fields
	if relatedTable, ok := generic["relatedTable"].(string); ok && relatedTable != "" {
		model.RelatedTableID = types.StringValue(relatedTable)
	}
	if relationType, ok := generic["relationType"].(string); ok && relationType != "" {
		model.RelationType = types.StringValue(relationType)
	}
	if twoWay, ok := generic["twoWay"].(bool); ok {
		if !model.TwoWay.IsNull() {
			model.TwoWay = types.BoolValue(twoWay)
		}
	}
	if twoWayKey, ok := generic["twoWayKey"].(string); ok && twoWayKey != "" {
		if !model.TwoWayKey.IsNull() {
			model.TwoWayKey = types.StringValue(twoWayKey)
		}
	}
	if onDelete, ok := generic["onDelete"].(string); ok && onDelete != "" {
		if !model.OnDelete.IsNull() {
			model.OnDelete = types.StringValue(onDelete)
		}
	}

	// Default value — normalize to string representation
	if defaultVal, ok := generic["default"]; ok && defaultVal != nil {
		switch v := defaultVal.(type) {
		case string:
			if v != "" {
				model.DefaultStr = types.StringValue(v)
			} else if !model.DefaultStr.IsNull() {
				model.DefaultStr = types.StringNull()
			}
		case bool:
			if !model.DefaultStr.IsNull() {
				model.DefaultStr = types.StringValue(fmt.Sprintf("%t", v))
			}
		case float64:
			if !model.DefaultStr.IsNull() {
				if model.Type.ValueString() == "integer" {
					model.DefaultStr = types.StringValue(fmt.Sprintf("%d", int64(v)))
				} else {
					model.DefaultStr = types.StringValue(fmt.Sprintf("%g", v))
				}
			}
		}
	}
}
