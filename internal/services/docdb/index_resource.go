package docdb

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &indexResource{}
	_ resource.ResourceWithConfigure   = &indexResource{}
	_ resource.ResourceWithImportState = &indexResource{}
)

// indexResource manages one index on a collection. Indexes have no update
// route, so every configurable attribute forces replacement.
type indexResource struct {
	product Product
	clients *common.AppwriteClients
}

type indexResourceModel struct {
	ID           types.String `tfsdk:"id"`
	DatabaseID   types.String `tfsdk:"database_id"`
	CollectionID types.String `tfsdk:"collection_id"`
	Key          types.String `tfsdk:"key"`
	Type         types.String `tfsdk:"type"`
	Attributes   types.List   `tfsdk:"attributes"`
	Orders       types.List   `tfsdk:"orders"`
	Lengths      types.List   `tfsdk:"lengths"`

	Status    types.String `tfsdk:"status"`
	Error     types.String `tfsdk:"error"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	ProjectID types.String `tfsdk:"project_id"`
}

// NewIndexResource returns a constructor for the index resource of one product.
func NewIndexResource(product Product) func() resource.Resource {
	return func() resource.Resource {
		return &indexResource{product: product}
	}
}

func (r *indexResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_index", req.ProviderTypeName, r.product)
}

func (r *indexResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf(
			"Manages an index on a collection in an Appwrite %s database. Indexes have no update route, so changing any "+
				"argument replaces the index. Creation is asynchronous; Terraform waits for the index to become available so a "+
				"dependent resource is not handed one that is still building.",
			r.product.Label(),
		),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The index identifier, in the form `database_id/collection_id/key`.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"database_id": schema.StringAttribute{
				Description:   "The database ID.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"collection_id": schema.StringAttribute{
				Description:   "The collection ID the index is created on.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key": schema.StringAttribute{
				Description:   "The index key, unique within the collection.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				Description:   "The index type, for example `key`, `unique` or `fulltext`. The accepted values depend on the product and the attribute being indexed.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"attributes": schema.ListAttribute{
				Description:   "The document attributes to index, in order.",
				Required:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"orders": schema.ListAttribute{
				Description:   "The sort order per attribute, `ASC` or `DESC`. Positional, matching `attributes`.",
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace(), listplanmodifier.UseStateForUnknown()},
			},
			"lengths": schema.ListAttribute{
				Description:   "The indexed prefix length per attribute. Positional, matching `attributes`.",
				Optional:      true,
				Computed:      true,
				ElementType:   types.Int64Type,
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace(), listplanmodifier.UseStateForUnknown()},
			},

			"status": schema.StringAttribute{Description: "The index status: `available`, `processing`, `deleting`, `stuck` or `failed`.", Computed: true},
			"error":  schema.StringAttribute{Description: "The error reported when the index failed to build.", Computed: true},

			"created_at": schema.StringAttribute{Description: "The index creation timestamp in ISO 8601 format.", Computed: true},
			"updated_at": schema.StringAttribute{
				Description:   "The index last update timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{common.UseStateForUnknownUnlessUpdating()},
			},
			"project_id": common.ProjectIDAttribute(),
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
	r.clients = clients
}

func (r *indexResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan indexResourceModel
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

	var attributes []string
	resp.Diagnostics.Append(plan.Attributes.ElementsAs(ctx, &attributes, false)...)

	var orders []string
	if !plan.Orders.IsNull() && !plan.Orders.IsUnknown() {
		resp.Diagnostics.Append(plan.Orders.ElementsAs(ctx, &orders, false)...)
	}
	var lengths []int
	if !plan.Lengths.IsNull() && !plan.Lengths.IsUnknown() {
		resp.Diagnostics.Append(plan.Lengths.ElementsAs(ctx, &lengths, false)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	databaseID := plan.DatabaseID.ValueString()
	collectionID := plan.CollectionID.ValueString()
	key := plan.Key.ValueString()

	if _, err := api.CreateIndex(databaseID, collectionID, key, plan.Type.ValueString(), attributes, IndexOptions{
		Orders:  orders,
		Lengths: lengths,
	}); err != nil {
		resp.Diagnostics.AddError("Error creating index", common.FormatErrorWithAuthGuidance(err, common.DatabaseProductGuidance(fmt.Sprintf("appwrite_%s_index", r.product), "collections.write")))
		return
	}

	// The index now exists remotely. Persist its identity before waiting, so a
	// wait that times out or reports a failed build leaves a resource Terraform
	// can refresh or destroy rather than an index it has forgotten about.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_id"), databaseID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("collection_id"), collectionID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), key)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), indexID(databaseID, collectionID, key))...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Index builds are asynchronous. Waiting keeps a dependent resource from
	// querying an index that is still processing.
	if err := common.WaitForColumnAvailable(ctx, func() (interface{}, error) {
		index, err := api.GetIndex(databaseID, collectionID, key)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": index.Status}, nil
	}, key); err != nil {
		resp.Diagnostics.AddError("Error waiting for index", err.Error())
		return
	}

	index, err := api.GetIndex(databaseID, collectionID, key)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created index", common.FormatErrorWithAuthGuidance(err, common.DatabaseProductGuidance(fmt.Sprintf("appwrite_%s_index", r.product), "collections.write")))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, index, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *indexResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state indexResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}

	index, err := apiFor(r.clients, r.product, projectID).
		GetIndex(state.DatabaseID.ValueString(), state.CollectionID.ValueString(), state.Key.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading index", common.FormatErrorWithAuthGuidance(err, common.DatabaseProductGuidance(fmt.Sprintf("appwrite_%s_index", r.product), "collections.write")))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, index, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is unreachable: every configurable attribute forces replacement.
func (r *indexResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan indexResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *indexResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state indexResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}

	err = apiFor(r.clients, r.product, projectID).
		DeleteIndex(state.DatabaseID.ValueString(), state.CollectionID.ValueString(), state.Key.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting index", common.FormatErrorWithAuthGuidance(err, common.DatabaseProductGuidance(fmt.Sprintf("appwrite_%s_index", r.product), "collections.write")))
	}
}

func (r *indexResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, ok := splitImportID(req.ID, 3)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: database_id/collection_id/key, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("collection_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), indexID(parts[0], parts[1], parts[2]))...)
}

func (r *indexResource) mapToState(ctx context.Context, index *models.Index, model *indexResourceModel, diagnostics *diag.Diagnostics) {
	model.Key = types.StringValue(index.Key)
	model.Type = types.StringValue(index.Type)
	model.Status = types.StringValue(index.Status)
	model.Error = types.StringValue(index.Error)
	model.CreatedAt = types.StringValue(index.CreatedAt)
	model.UpdatedAt = types.StringValue(index.UpdatedAt)
	model.ID = types.StringValue(indexID(
		model.DatabaseID.ValueString(), model.CollectionID.ValueString(), index.Key))

	attributes, diags := types.ListValueFrom(ctx, types.StringType, nonNilStrings(index.Attributes))
	diagnostics.Append(diags...)
	model.Attributes = attributes

	orders, diags := types.ListValueFrom(ctx, types.StringType, nonNilStrings(index.Orders))
	diagnostics.Append(diags...)
	model.Orders = orders

	lengths, diags := types.ListValueFrom(ctx, types.Int64Type, nonNilInts(index.Lengths))
	diagnostics.Append(diags...)
	model.Lengths = lengths
}

func indexID(databaseID, collectionID, key string) string {
	return fmt.Sprintf("%s/%s/%s", databaseID, collectionID, key)
}
