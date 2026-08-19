package docdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/id"
	"github.com/appwrite/sdk-for-go/v7/models"
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
	_ resource.Resource                = &documentResource{}
	_ resource.ResourceWithConfigure   = &documentResource{}
	_ resource.ResourceWithImportState = &documentResource{}
)

type documentResource struct {
	product Product
	clients *common.AppwriteClients
}

type documentResourceModel struct {
	ID           types.String `tfsdk:"id"`
	DatabaseID   types.String `tfsdk:"database_id"`
	CollectionID types.String `tfsdk:"collection_id"`
	Data         types.String `tfsdk:"data"`
	Permissions  types.Set    `tfsdk:"permissions"`

	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	ProjectID types.String `tfsdk:"project_id"`
}

// NewDocumentResource returns a constructor for the document resource of one
// product.
func NewDocumentResource(product Product) func() resource.Resource {
	return func() resource.Resource {
		return &documentResource{product: product}
	}
}

func (r *documentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_document", req.ProviderTypeName, r.product)
}

func (r *documentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	dataDescription := "The document body as a JSON object string, for example `jsonencode({ title = \"Hello\" })`."
	if r.product == ProductVectorsDB {
		dataDescription += " A VectorsDB document carries its embedding, which must have exactly the collection's `dimension` values."
	}
	dataDescription += " Only the keys present here are tracked, so fields written by other clients do not show as drift."

	resp.Schema = schema.Schema{
		Description: fmt.Sprintf(
			"Manages a document in an Appwrite %s collection.\n\n"+
				"This manages data rather than infrastructure, so it suits seed and reference records. Documents written by "+
				"an application at runtime should not be managed here.",
			r.product.Label(),
		),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The document ID. Generated when omitted.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"database_id": schema.StringAttribute{
				Description:   "The database ID.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"collection_id": schema.StringAttribute{
				Description:   "The collection ID the document belongs to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"data": schema.StringAttribute{
				Description: dataDescription,
				Required:    true,
			},
			"permissions": schema.SetAttribute{
				Description: "The document permissions. Only enforced when the collection has `document_security` enabled.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},

			"created_at": schema.StringAttribute{Description: "The document creation timestamp in ISO 8601 format.", Computed: true},
			"updated_at": schema.StringAttribute{
				Description:   "The document last update timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{common.UseStateForUnknownUnlessUpdating()},
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

func (r *documentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *documentResource) permissions(ctx context.Context, plan documentResourceModel, diagnostics *diag.Diagnostics) []string {
	if plan.Permissions.IsNull() || plan.Permissions.IsUnknown() {
		return nil
	}
	var permissions []string
	diagnostics.Append(plan.Permissions.ElementsAs(ctx, &permissions, false)...)
	if permissions == nil {
		permissions = []string{}
	}
	return permissions
}

func (r *documentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan documentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(plan.Data.ValueString()), &data); err != nil {
		resp.Diagnostics.AddError("Invalid document data", fmt.Sprintf("data must be a JSON object: %s", err))
		return
	}

	documentID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		documentID = id.Unique()
	}

	permissions := r.permissions(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	document, err := apiFor(r.clients, r.product, projectID).
		CreateDocument(plan.DatabaseID.ValueString(), plan.CollectionID.ValueString(), documentID, data, permissions)
	if err != nil {
		resp.Diagnostics.AddError("Error creating document", common.FormatErrorWithAuthGuidance(err, common.DatabaseProductGuidance(fmt.Sprintf("appwrite_%s_document", r.product), "documents.write")))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, document, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *documentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state documentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}

	document, err := apiFor(r.clients, r.product, projectID).
		GetDocument(state.DatabaseID.ValueString(), state.CollectionID.ValueString(), state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading document", common.FormatErrorWithAuthGuidance(err, common.DatabaseProductGuidance(fmt.Sprintf("appwrite_%s_document", r.product), "documents.write")))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, document, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *documentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan documentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(plan.Data.ValueString()), &data); err != nil {
		resp.Diagnostics.AddError("Invalid document data", fmt.Sprintf("data must be a JSON object: %s", err))
		return
	}

	permissions := r.permissions(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	document, err := apiFor(r.clients, r.product, projectID).
		UpdateDocument(plan.DatabaseID.ValueString(), plan.CollectionID.ValueString(), plan.ID.ValueString(), data, permissions)
	if err != nil {
		resp.Diagnostics.AddError("Error updating document", common.FormatErrorWithAuthGuidance(err, common.DatabaseProductGuidance(fmt.Sprintf("appwrite_%s_document", r.product), "documents.write")))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(ctx, document, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *documentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state documentResourceModel
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
		DeleteDocument(state.DatabaseID.ValueString(), state.CollectionID.ValueString(), state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting document", common.FormatErrorWithAuthGuidance(err, common.DatabaseProductGuidance(fmt.Sprintf("appwrite_%s_document", r.product), "documents.write")))
	}
}

func (r *documentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, ok := splitImportID(req.ID, 3)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: database_id/collection_id/document_id, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("collection_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

func (r *documentResource) mapToState(ctx context.Context, document *models.Document, model *documentResourceModel, diagnostics *diag.Diagnostics) {
	model.ID = types.StringValue(document.Id)
	model.DatabaseID = types.StringValue(document.DatabaseId)
	model.CollectionID = types.StringValue(document.CollectionId)
	model.CreatedAt = types.StringValue(document.CreatedAt)
	model.UpdatedAt = types.StringValue(document.UpdatedAt)

	permissions, diags := types.SetValueFrom(ctx, types.StringType, nonNilStrings(document.Permissions))
	diagnostics.Append(diags...)
	model.Permissions = permissions

	// The typed model carries no body, so the full response is decoded and the
	// $-prefixed system fields dropped.
	//
	// Values are kept as raw JSON rather than decoded into interface{}: doing
	// the latter turns every number into a float64, which silently rounds an
	// integer beyond 2^53 and would rewrite it in state on each refresh.
	var raw map[string]json.RawMessage
	if err := document.Decode(&raw); err != nil {
		diagnostics.AddError("Error decoding document data", err.Error())
		return
	}

	// Track only the keys the configuration mentions. A collection may hold
	// fields written by the application, and adopting those would show as drift
	// on every plan.
	var configured map[string]json.RawMessage
	if !model.Data.IsNull() && !model.Data.IsUnknown() {
		if err := json.Unmarshal([]byte(model.Data.ValueString()), &configured); err != nil {
			configured = nil
		}
	}

	filtered := make(map[string]json.RawMessage, len(raw))
	for key, value := range raw {
		if strings.HasPrefix(key, "$") {
			continue
		}
		// On import there is no prior configuration, so everything is adopted.
		if configured != nil {
			if _, ok := configured[key]; !ok {
				continue
			}
		}
		// Whitespace in the response would otherwise land in state and read as
		// a difference against the compact form jsonencode produces.
		var compact bytes.Buffer
		if err := json.Compact(&compact, value); err != nil {
			diagnostics.AddError("Error normalising document data", err.Error())
			return
		}
		filtered[key] = json.RawMessage(compact.Bytes())
	}

	encoded, err := json.Marshal(filtered)
	if err != nil {
		diagnostics.AddError("Error encoding document data", err.Error())
		return
	}
	model.Data = types.StringValue(string(encoded))
}
