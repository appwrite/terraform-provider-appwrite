package docdb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/id"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &collectionResource{}
	_ resource.ResourceWithConfigure   = &collectionResource{}
	_ resource.ResourceWithImportState = &collectionResource{}
)

type collectionResource struct {
	product Product
	clients *common.AppwriteClients
}

type collectionResourceModel struct {
	ID               types.String `tfsdk:"id"`
	DatabaseID       types.String `tfsdk:"database_id"`
	Name             types.String `tfsdk:"name"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	DocumentSecurity types.Bool   `tfsdk:"document_security"`
	Permissions      types.Set    `tfsdk:"permissions"`
	Dimension        types.Int64  `tfsdk:"dimension"`
	Attributes       types.String `tfsdk:"attributes"`

	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	ProjectID types.String `tfsdk:"project_id"`
}

// NewCollectionResource returns a constructor for the collection resource of
// one product.
func NewCollectionResource(product Product) func() resource.Resource {
	return func() resource.Resource {
		return &collectionResource{product: product}
	}
}

func (r *collectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_collection", req.ProviderTypeName, r.product)
}

func (r *collectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description:   "The collection ID. Must be unique within the database. Generated when omitted.",
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
		},
		"database_id": schema.StringAttribute{
			Description:   "The database ID the collection belongs to.",
			Required:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"name": schema.StringAttribute{
			Description: "The collection name.",
			Required:    true,
		},
		"enabled": schema.BoolAttribute{
			Description: "Whether the collection is enabled. When disabled it is inaccessible to users but still reachable with an API key. Defaults to true.",
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(true),
		},
		"document_security": schema.BoolAttribute{
			Description: "Whether document-level permissions are enforced in addition to collection-level ones.",
			Optional:    true,
			Computed:    true,
		},
		"permissions": schema.SetAttribute{
			Description: "The collection permissions, for example `read(\"any\")` or `write(\"users\")`.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
		},

		"created_at": schema.StringAttribute{
			Description: "The collection creation timestamp in ISO 8601 format.",
			Computed:    true,
		},
		"updated_at": schema.StringAttribute{
			Description:   "The collection last update timestamp in ISO 8601 format.",
			Computed:      true,
			PlanModifiers: []planmodifier.String{common.UseStateForUnknownUnlessUpdating()},
		},
		"project_id": common.ProjectIDAttribute(),
	}

	// Attributes can only be declared when the collection is created: there is
	// no route to add, change or remove one afterwards. Changing them therefore
	// replaces the collection, and the value is never refreshed from the server
	// -- the API returns its own normalised form with fields the configuration
	// never mentioned, which would read as drift on every plan.
	if r.product.SupportsAttributes() {
		attributes["attributes"] = schema.StringAttribute{
			Description: "Typed attribute definitions as a JSON array string, for example " +
				"`jsonencode([{ key = \"slug\", type = \"string\", size = 255, required = true }])`. " +
				"Applied only when the collection is created, so changing this replaces the collection. " +
				"An index can only be built on a declared attribute, so declare here anything you intend to index. " +
				"Not refreshed from the server, so drift on it is not detected.",
			Optional:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		}
	} else {
		attributes["attributes"] = schema.StringAttribute{
			Description: "Not applicable to VectorsDB collections, which take no typed attribute definitions. Present only so both products share one state shape.",
			Computed:    true,
		}
	}

	// Only VectorsDB collections carry an embedding dimension. Exposing it on
	// DocumentsDB would let a user configure something the route does not take.
	if r.product.SupportsDimension() {
		attributes["dimension"] = schema.Int64Attribute{
			Description: "The embedding dimension every vector in this collection must have. Required, and must match the model producing the embeddings.",
			Required:    true,
			Validators:  []validator.Int64{int64validator.AtLeast(1)},
		}
	} else {
		attributes["dimension"] = schema.Int64Attribute{
			Description: "Not applicable to DocumentsDB collections; always 0. Present only so both products share one state shape.",
			Computed:    true,
		}
	}

	description := fmt.Sprintf("Manages a collection in an Appwrite %s database.", r.product.Label())
	if r.product == ProductVectorsDB {
		description += " A VectorsDB collection stores embeddings of a fixed `dimension` and is searched by vector similarity."
	} else {
		description += " A DocumentsDB collection stores schemaless JSON documents."
	}

	resp.Schema = schema.Schema{Description: description, Attributes: attributes}
}

func (r *collectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// options builds the request. sendDimension is false on update unless the
// dimension actually changed: it is required in configuration, so it is always
// present in the plan, and resending it on an unrelated rename would put a
// re-index request in front of the server for no reason.
func (r *collectionResource) options(ctx context.Context, plan collectionResourceModel, sendDimension bool, diagnostics *diag.Diagnostics) CollectionOptions {
	var permissions []string
	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		diagnostics.Append(plan.Permissions.ElementsAs(ctx, &permissions, false)...)
		if permissions == nil {
			permissions = []string{}
		}
	}

	opts := CollectionOptions{
		Permissions:      permissions,
		DocumentSecurity: optBool(plan.DocumentSecurity),
		Enabled:          optBool(plan.Enabled),
	}
	if r.product.SupportsDimension() && sendDimension {
		opts.Dimension = optInt(plan.Dimension)
	}
	if r.product.SupportsAttributes() && !plan.Attributes.IsNull() && !plan.Attributes.IsUnknown() {
		var declared []interface{}
		if err := json.Unmarshal([]byte(plan.Attributes.ValueString()), &declared); err != nil {
			diagnostics.AddError("Invalid attributes", fmt.Sprintf("attributes must be a JSON array: %s", err))
			return opts
		}
		opts.Attributes = declared
	}
	return opts
}

func (r *collectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan collectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}

	collectionID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		collectionID = id.Unique()
	}

	opts := r.options(ctx, plan, true, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	collection, err := apiFor(r.clients, r.product, projectID).
		CreateCollection(plan.DatabaseID.ValueString(), collectionID, plan.Name.ValueString(), opts)
	if err != nil {
		resp.Diagnostics.AddError("Error creating collection", common.FormatErrorWithAuthGuidance(err, common.DatabaseProductGuidance(fmt.Sprintf("appwrite_%s_collection", r.product), "collections.write")))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, collection, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *collectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state collectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}

	collection, err := apiFor(r.clients, r.product, projectID).
		GetCollection(state.DatabaseID.ValueString(), state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading collection", common.FormatErrorWithAuthGuidance(err, common.DatabaseProductGuidance(fmt.Sprintf("appwrite_%s_collection", r.product), "collections.write")))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, collection, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *collectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state collectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}

	dimensionChanged := !plan.Dimension.Equal(state.Dimension)
	opts := r.options(ctx, plan, dimensionChanged, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	collection, err := apiFor(r.clients, r.product, projectID).
		UpdateCollection(plan.DatabaseID.ValueString(), plan.ID.ValueString(), plan.Name.ValueString(), opts)
	if err != nil {
		resp.Diagnostics.AddError("Error updating collection", common.FormatErrorWithAuthGuidance(err, common.DatabaseProductGuidance(fmt.Sprintf("appwrite_%s_collection", r.product), "collections.write")))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, collection, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *collectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state collectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}

	err = apiFor(r.clients, r.product, projectID).DeleteCollection(state.DatabaseID.ValueString(), state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting collection", common.FormatErrorWithAuthGuidance(err, common.DatabaseProductGuidance(fmt.Sprintf("appwrite_%s_collection", r.product), "collections.write")))
	}
}

func (r *collectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, ok := splitImportID(req.ID, 2)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: database_id/collection_id, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *collectionResource) mapToState(ctx context.Context, collection *Collection, model *collectionResourceModel, diagnostics *diag.Diagnostics) {
	model.ID = types.StringValue(collection.ID)
	model.DatabaseID = types.StringValue(collection.DatabaseID)
	model.Name = types.StringValue(collection.Name)
	model.Enabled = types.BoolValue(collection.Enabled)
	model.DocumentSecurity = types.BoolValue(collection.DocumentSecurity)
	model.Dimension = types.Int64Value(int64(collection.Dimension))
	// attributes is deliberately not mapped: it is create-only and the server
	// returns a normalised form that would read as drift.
	if !r.product.SupportsAttributes() {
		model.Attributes = types.StringNull()
	}
	model.CreatedAt = types.StringValue(collection.CreatedAt)
	model.UpdatedAt = types.StringValue(collection.UpdatedAt)

	permissions := collection.Permissions
	if permissions == nil {
		permissions = []string{}
	}
	set, diags := types.SetValueFrom(ctx, types.StringType, permissions)
	diagnostics.Append(diags...)
	model.Permissions = set
}
