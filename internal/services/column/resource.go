package column

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/appwrite/sdk-for-go/v3/appwrite"
	"github.com/appwrite/sdk-for-go/v3/id"
	"github.com/appwrite/sdk-for-go/v3/tablesdb"
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

const (
	colTypeInteger = "integer"
	colTypeFloat   = "float"
	colTypeString  = "string"
	colTypeEnum    = "enum"
	colTypeEmail   = "email"
	colTypeURL     = "url"
	colTypeLine    = "line"
	colTypeBigInt  = "bigint"
)

var allColumnTypes = "varchar, text, longtext, mediumtext, integer, bigint, float, boolean, enum, email, datetime, url, ip, point, line, polygon, relationship, string"

type columnResource struct {
	clients *common.AppwriteClients
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
	ProjectID      types.String  `tfsdk:"project_id"`
}

func NewColumnResource() resource.Resource {
	return &columnResource{}
}

func (r *columnResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tablesdb_column"
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
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
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
				Description: "Whether the column is an array. Applies to string, varchar, text, longtext, mediumtext, integer, bigint, float, boolean, enum, email, datetime, url, ip types.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"size": schema.Int64Attribute{
				Description: "Maximum length. Required for string and varchar types.",
				Optional:    true,
			},
			"min": schema.Int64Attribute{
				Description: "Minimum value. Applies to integer and bigint types.",
				Optional:    true,
			},
			"max": schema.Int64Attribute{
				Description: "Maximum value. Applies to integer and bigint types.",
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The column last update timestamp in ISO 8601 format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": common.ProjectIDAttribute(),
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
	r.clients = clients
}

func (r *columnResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan columnResourceModel
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

	databaseID := plan.DatabaseID.ValueString()
	tableID := plan.TableID.ValueString()
	key := plan.Key.ValueString()
	if plan.Key.IsNull() || plan.Key.IsUnknown() {
		key = id.Unique()
	}
	required := plan.Required.ValueBool()
	columnType := plan.Type.ValueString()
	array := plan.Array.ValueBool()

	var responseJSON []byte

	switch columnType {
	case colTypeString:
		var opts []tablesdb.CreateTextColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateTextColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, tablesdbClient.WithCreateTextColumnArray(array))
		if !plan.Encrypt.IsNull() && !plan.Encrypt.IsUnknown() {
			opts = append(opts, tablesdbClient.WithCreateTextColumnEncrypt(plan.Encrypt.ValueBool()))
		}
		col, e := tablesdbClient.CreateTextColumn(databaseID, tableID, key, required, opts...)
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
			opts = append(opts, tablesdbClient.WithCreateVarcharColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, tablesdbClient.WithCreateVarcharColumnArray(array))
		if !plan.Encrypt.IsNull() && !plan.Encrypt.IsUnknown() {
			opts = append(opts, tablesdbClient.WithCreateVarcharColumnEncrypt(plan.Encrypt.ValueBool()))
		}
		col, e := tablesdbClient.CreateVarcharColumn(databaseID, tableID, key, int(plan.Size.ValueInt64()), required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "text":
		var opts []tablesdb.CreateTextColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateTextColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, tablesdbClient.WithCreateTextColumnArray(array))
		if !plan.Encrypt.IsNull() && !plan.Encrypt.IsUnknown() {
			opts = append(opts, tablesdbClient.WithCreateTextColumnEncrypt(plan.Encrypt.ValueBool()))
		}
		col, e := tablesdbClient.CreateTextColumn(databaseID, tableID, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "longtext":
		var opts []tablesdb.CreateLongtextColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateLongtextColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, tablesdbClient.WithCreateLongtextColumnArray(array))
		if !plan.Encrypt.IsNull() && !plan.Encrypt.IsUnknown() {
			opts = append(opts, tablesdbClient.WithCreateLongtextColumnEncrypt(plan.Encrypt.ValueBool()))
		}
		col, e := tablesdbClient.CreateLongtextColumn(databaseID, tableID, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "mediumtext":
		var opts []tablesdb.CreateMediumtextColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateMediumtextColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, tablesdbClient.WithCreateMediumtextColumnArray(array))
		if !plan.Encrypt.IsNull() && !plan.Encrypt.IsUnknown() {
			opts = append(opts, tablesdbClient.WithCreateMediumtextColumnEncrypt(plan.Encrypt.ValueBool()))
		}
		col, e := tablesdbClient.CreateMediumtextColumn(databaseID, tableID, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case colTypeInteger:
		var opts []tablesdb.CreateIntegerColumnOption
		if !plan.Min.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateIntegerColumnMin(int(plan.Min.ValueInt64())))
		}
		if !plan.Max.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateIntegerColumnMax(int(plan.Max.ValueInt64())))
		}
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateIntegerColumnDefault(parseIntDefault(plan.DefaultStr.ValueString())))
		}
		opts = append(opts, tablesdbClient.WithCreateIntegerColumnArray(array))
		col, e := tablesdbClient.CreateIntegerColumn(databaseID, tableID, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case colTypeBigInt:
		var opts []tablesdb.CreateBigIntColumnOption
		if !plan.Min.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateBigIntColumnMin(int(plan.Min.ValueInt64())))
		}
		if !plan.Max.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateBigIntColumnMax(int(plan.Max.ValueInt64())))
		}
		if !plan.DefaultStr.IsNull() {
			def, parseErr := parseBigIntDefault(plan.DefaultStr.ValueString())
			if parseErr != nil {
				resp.Diagnostics.AddError("Invalid default value", parseErr.Error())
				return
			}
			opts = append(opts, tablesdbClient.WithCreateBigIntColumnDefault(def))
		}
		opts = append(opts, tablesdbClient.WithCreateBigIntColumnArray(array))
		col, e := tablesdbClient.CreateBigIntColumn(databaseID, tableID, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case colTypeFloat:
		var opts []tablesdb.CreateFloatColumnOption
		if !plan.FloatMin.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateFloatColumnMin(plan.FloatMin.ValueFloat64()))
		}
		if !plan.FloatMax.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateFloatColumnMax(plan.FloatMax.ValueFloat64()))
		}
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateFloatColumnDefault(parseFloatDefault(plan.DefaultStr.ValueString())))
		}
		opts = append(opts, tablesdbClient.WithCreateFloatColumnArray(array))
		col, e := tablesdbClient.CreateFloatColumn(databaseID, tableID, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "boolean":
		var opts []tablesdb.CreateBooleanColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateBooleanColumnDefault(plan.DefaultStr.ValueString() == "true"))
		}
		opts = append(opts, tablesdbClient.WithCreateBooleanColumnArray(array))
		col, e := tablesdbClient.CreateBooleanColumn(databaseID, tableID, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case colTypeEnum:
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
			opts = append(opts, tablesdbClient.WithCreateEnumColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, tablesdbClient.WithCreateEnumColumnArray(array))
		col, e := tablesdbClient.CreateEnumColumn(databaseID, tableID, key, elements, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case colTypeEmail:
		var opts []tablesdb.CreateEmailColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateEmailColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, tablesdbClient.WithCreateEmailColumnArray(array))
		col, e := tablesdbClient.CreateEmailColumn(databaseID, tableID, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "datetime":
		var opts []tablesdb.CreateDatetimeColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateDatetimeColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, tablesdbClient.WithCreateDatetimeColumnArray(array))
		col, e := tablesdbClient.CreateDatetimeColumn(databaseID, tableID, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case colTypeURL:
		var opts []tablesdb.CreateUrlColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateUrlColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, tablesdbClient.WithCreateUrlColumnArray(array))
		col, e := tablesdbClient.CreateUrlColumn(databaseID, tableID, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "ip":
		var opts []tablesdb.CreateIpColumnOption
		if !plan.DefaultStr.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateIpColumnDefault(plan.DefaultStr.ValueString()))
		}
		opts = append(opts, tablesdbClient.WithCreateIpColumnArray(array))
		col, e := tablesdbClient.CreateIpColumn(databaseID, tableID, key, required, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "point":
		col, e := tablesdbClient.CreatePointColumn(databaseID, tableID, key, required)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case colTypeLine:
		col, e := tablesdbClient.CreateLineColumn(databaseID, tableID, key, required)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "polygon":
		col, e := tablesdbClient.CreatePolygonColumn(databaseID, tableID, key, required)
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
			opts = append(opts, tablesdbClient.WithCreateRelationshipColumnTwoWay(plan.TwoWay.ValueBool()))
		}
		if !plan.Key.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateRelationshipColumnKey(key))
		}
		if !plan.TwoWayKey.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateRelationshipColumnTwoWayKey(plan.TwoWayKey.ValueString()))
		}
		if !plan.OnDelete.IsNull() {
			opts = append(opts, tablesdbClient.WithCreateRelationshipColumnOnDelete(plan.OnDelete.ValueString()))
		}
		col, e := tablesdbClient.CreateRelationshipColumn(databaseID, tableID, plan.RelatedTableID.ValueString(), plan.RelationType.ValueString(), opts...)
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

	if err := common.WaitForColumnAvailable(ctx, func() (interface{}, error) {
		return common.GetColumnRaw(r.clients.ClientForProject(projectID), databaseID, tableID, key)
	}, key); err != nil {
		resp.Diagnostics.AddError("Error waiting for column to become available", err.Error())
		return
	}

	r.readResponseIntoState(ctx, responseJSON, &plan, &resp.Diagnostics)
	plan.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *columnResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state columnResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	sdkClient := r.clients.ClientForProject(projectID)

	generic, err := common.GetColumnRaw(sdkClient, state.DatabaseID.ValueString(), state.TableID.ValueString(), state.Key.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading column", common.FormatError(err))
		return
	}

	responseJSON, _ := json.Marshal(generic)
	r.readResponseIntoState(ctx, responseJSON, &state, &resp.Diagnostics)
	state.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *columnResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan columnResourceModel
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

	databaseID := plan.DatabaseID.ValueString()
	tableID := plan.TableID.ValueString()
	key := plan.Key.ValueString()
	required := plan.Required.ValueBool()
	columnType := plan.Type.ValueString()

	defaultStr := ""
	if !plan.DefaultStr.IsNull() {
		defaultStr = plan.DefaultStr.ValueString()
	}

	var responseJSON []byte

	switch columnType {
	case colTypeString:
		var opts []tablesdb.UpdateTextColumnOption
		col, e := tablesdbClient.UpdateTextColumn(databaseID, tableID, key, required, defaultStr, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "varchar":
		var opts []tablesdb.UpdateVarcharColumnOption
		if !plan.Size.IsNull() {
			opts = append(opts, tablesdbClient.WithUpdateVarcharColumnSize(int(plan.Size.ValueInt64())))
		}
		col, e := tablesdbClient.UpdateVarcharColumn(databaseID, tableID, key, required, defaultStr, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "text":
		col, e := tablesdbClient.UpdateTextColumn(databaseID, tableID, key, required, defaultStr)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "longtext":
		col, e := tablesdbClient.UpdateLongtextColumn(databaseID, tableID, key, required, defaultStr)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "mediumtext":
		col, e := tablesdbClient.UpdateMediumtextColumn(databaseID, tableID, key, required, defaultStr)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case colTypeInteger:
		var opts []tablesdb.UpdateIntegerColumnOption
		if !plan.Min.IsNull() {
			opts = append(opts, tablesdbClient.WithUpdateIntegerColumnMin(int(plan.Min.ValueInt64())))
		}
		if !plan.Max.IsNull() {
			opts = append(opts, tablesdbClient.WithUpdateIntegerColumnMax(int(plan.Max.ValueInt64())))
		}
		col, e := tablesdbClient.UpdateIntegerColumn(databaseID, tableID, key, required, parseIntDefault(defaultStr), opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case colTypeBigInt:
		var opts []tablesdb.UpdateBigIntColumnOption
		if !plan.Min.IsNull() {
			opts = append(opts, tablesdbClient.WithUpdateBigIntColumnMin(int(plan.Min.ValueInt64())))
		}
		if !plan.Max.IsNull() {
			opts = append(opts, tablesdbClient.WithUpdateBigIntColumnMax(int(plan.Max.ValueInt64())))
		}
		def, parseErr := parseBigIntDefault(defaultStr)
		if parseErr != nil {
			resp.Diagnostics.AddError("Invalid default value", parseErr.Error())
			return
		}
		col, e := tablesdbClient.UpdateBigIntColumn(databaseID, tableID, key, required, def, opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case colTypeFloat:
		var opts []tablesdb.UpdateFloatColumnOption
		if !plan.FloatMin.IsNull() {
			opts = append(opts, tablesdbClient.WithUpdateFloatColumnMin(plan.FloatMin.ValueFloat64()))
		}
		if !plan.FloatMax.IsNull() {
			opts = append(opts, tablesdbClient.WithUpdateFloatColumnMax(plan.FloatMax.ValueFloat64()))
		}
		col, e := tablesdbClient.UpdateFloatColumn(databaseID, tableID, key, required, parseFloatDefault(defaultStr), opts...)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "boolean":
		col, e := tablesdbClient.UpdateBooleanColumn(databaseID, tableID, key, required, defaultStr == "true")
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case colTypeEnum:
		var elements []string
		resp.Diagnostics.Append(plan.Elements.ElementsAs(ctx, &elements, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		col, e := tablesdbClient.UpdateEnumColumn(databaseID, tableID, key, elements, required, defaultStr)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case colTypeEmail:
		col, e := tablesdbClient.UpdateEmailColumn(databaseID, tableID, key, required, defaultStr)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "datetime":
		col, e := tablesdbClient.UpdateDatetimeColumn(databaseID, tableID, key, required, defaultStr)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case colTypeURL:
		col, e := tablesdbClient.UpdateUrlColumn(databaseID, tableID, key, required, defaultStr)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "ip":
		col, e := tablesdbClient.UpdateIpColumn(databaseID, tableID, key, required, defaultStr)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "point":
		col, e := tablesdbClient.UpdatePointColumn(databaseID, tableID, key, required)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case colTypeLine:
		col, e := tablesdbClient.UpdateLineColumn(databaseID, tableID, key, required)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "polygon":
		col, e := tablesdbClient.UpdatePolygonColumn(databaseID, tableID, key, required)
		err = e
		if col != nil {
			responseJSON, _ = json.Marshal(col)
		}

	case "relationship":
		var opts []tablesdb.UpdateRelationshipColumnOption
		if !plan.OnDelete.IsNull() {
			opts = append(opts, tablesdbClient.WithUpdateRelationshipColumnOnDelete(plan.OnDelete.ValueString()))
		}
		col, e := tablesdbClient.UpdateRelationshipColumn(databaseID, tableID, key, opts...)
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

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	tablesdbClient := appwrite.NewTablesDB(r.clients.ClientForProject(projectID))

	_, err = tablesdbClient.DeleteColumn(state.DatabaseID.ValueString(), state.TableID.ValueString(), state.Key.ValueString())
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
	// Don't overwrite type when already set — preserve the user's value.
	// During import, type is not yet in state, so populate it from the API.
	// The API returns internal names that differ from the schema in some cases
	// (e.g. "double" for "float", "linestring" for "line"), so we normalize.
	if model.Type.IsNull() || model.Type.IsUnknown() {
		if apiType, ok := generic["type"].(string); ok {
			model.Type = types.StringValue(normalizeAPIColumnType(apiType, generic))
		}
	}
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
	if minVal, ok := generic["min"].(float64); ok {
		columnType := model.Type.ValueString()
		if columnType == colTypeInteger || columnType == colTypeBigInt {
			model.Min = types.Int64Value(int64(minVal))
		} else if columnType == colTypeFloat {
			model.FloatMin = types.Float64Value(minVal)
		}
	}
	if maxVal, ok := generic["max"].(float64); ok {
		columnType := model.Type.ValueString()
		if columnType == colTypeInteger || columnType == colTypeBigInt {
			model.Max = types.Int64Value(int64(maxVal))
		} else if columnType == colTypeFloat {
			model.FloatMax = types.Float64Value(maxVal)
		}
	}
	if encrypt, ok := generic["encrypt"].(bool); ok {
		model.Encrypt = types.BoolValue(encrypt)
	} else if model.Encrypt.IsNull() || model.Encrypt.IsUnknown() {
		// The API only returns encrypt for string-like types. Default to false
		// for other types so the state matches the schema default during import.
		model.Encrypt = types.BoolValue(false)
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
			if !model.DefaultStr.IsNull() || model.DefaultStr.IsUnknown() {
				model.DefaultStr = types.StringValue(fmt.Sprintf("%t", v))
			}
		case float64:
			if !model.DefaultStr.IsNull() || model.DefaultStr.IsUnknown() {
				if model.Type.ValueString() == colTypeInteger || model.Type.ValueString() == colTypeBigInt {
					model.DefaultStr = types.StringValue(fmt.Sprintf("%d", int64(v)))
				} else {
					model.DefaultStr = types.StringValue(fmt.Sprintf("%g", v))
				}
			}
		}
	}
}

// normalizeAPIColumnType maps an Appwrite API type back to the user-facing
// schema type. Most types are returned as-is, but a few differ:
//
//	API "double"     → schema "float"
//	API "linestring" → schema "line"
//	API "string" with format "email"/"enum"/"url"/"ip" → that format name
func normalizeAPIColumnType(apiType string, response map[string]interface{}) string {
	switch apiType {
	case "double":
		return colTypeFloat
	case "linestring":
		return colTypeLine
	case colTypeString:
		if format, ok := response["format"].(string); ok {
			switch format {
			case colTypeEmail, colTypeEnum, colTypeURL, "ip":
				return format
			}
		}
		return colTypeString
	default:
		return apiType
	}
}
