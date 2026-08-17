package dedicated

import (
	"context"
	"fmt"
	"slices"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &extensionResource{}
	_ resource.ResourceWithConfigure   = &extensionResource{}
	_ resource.ResourceWithImportState = &extensionResource{}
)

// extensionResource manages one installed PostgreSQL extension. Extensions have
// no update route -- an extension is either installed or not -- so every
// attribute forces replacement.
type extensionResource struct {
	engine  Engine
	clients *common.AppwriteClients
}

type extensionResourceModel struct {
	ID         types.String `tfsdk:"id"`
	DatabaseID types.String `tfsdk:"database_id"`
	Name       types.String `tfsdk:"name"`
	ProjectID  types.String `tfsdk:"project_id"`
}

// NewExtensionResource returns a constructor for the extension resource.
func NewExtensionResource(engine Engine) func() resource.Resource {
	return func() resource.Resource {
		return &extensionResource{engine: engine}
	}
}

func (r *extensionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_extension", req.ProviderTypeName, r.engine)
}

func (r *extensionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf(
			"Installs an extension into a dedicated Appwrite %s database. Read the installable names from the "+
				"`available` list of the corresponding extensions data source.",
			r.engine.Label(),
		),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The extension identifier, in the form `database_id/name`.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"database_id": schema.StringAttribute{
				Description:   "The dedicated database ID to install the extension into.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description:   "The extension name, for example `postgis` or `pg_trgm`.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

func (r *extensionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// api resolves the project and returns the extension adapter for the engine.
func (r *extensionResource) api(resourceProjectID types.String) (extensionAPI, string, error) {
	projectID, err := common.ResolveProjectID(r.clients, resourceProjectID)
	if err != nil {
		return nil, "", err
	}
	api, ok := newExtensionAPI(r.engine, r.clients.ClientForProject(projectID))
	if !ok {
		return nil, "", fmt.Errorf("%s databases do not support extensions", r.engine.Label())
	}
	return api, projectID, nil
}

func (r *extensionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan extensionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	api, projectID, err := r.api(plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving extension client", err.Error())
		return
	}

	if _, err := api.CreateExtension(plan.DatabaseID.ValueString(), plan.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error installing extension", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	plan.ID = types.StringValue(extensionID(plan.DatabaseID.ValueString(), plan.Name.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *extensionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state extensionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	api, projectID, err := r.api(state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving extension client", err.Error())
		return
	}

	extensions, err := api.ListExtensions(state.DatabaseID.ValueString())
	if err != nil {
		// A deleted database takes its extensions with it.
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading extensions", common.FormatError(err))
		return
	}

	// There is no per-extension read route, so installation is confirmed
	// against the database's installed list.
	if !slices.Contains(extensions.Installed, state.Name.ValueString()) {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ProjectID = types.StringValue(projectID)
	state.ID = types.StringValue(extensionID(state.DatabaseID.ValueString(), state.Name.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is unreachable: every configurable attribute forces replacement.
func (r *extensionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan extensionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *extensionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state extensionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	api, _, err := r.api(state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving extension client", err.Error())
		return
	}

	if _, err := api.DeleteExtension(state.DatabaseID.ValueString(), state.Name.ValueString()); err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error removing extension", common.FormatError(err))
	}
}

func (r *extensionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	databaseID, name, ok := splitImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: database_id/name, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_id"), databaseID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), extensionID(databaseID, name))...)
}

func extensionID(databaseID, name string) string {
	return databaseID + "/" + name
}
