package project

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/models"
	sdkproject "github.com/appwrite/sdk-for-go/v7/project"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

// toggleItem is one switch in a project's API settings: a service or a
// protocol. Appwrite stores only the switches someone has set and reports every
// other one as enabled, so enabled is the default for all of them.
type toggleItem struct {
	// id is the identifier Appwrite uses in the route and in the project model,
	// and the attribute name in the schema.
	id string
	// name is how the item is referred to in its attribute description.
	name string
	// note, when set, is appended to the attribute description.
	note string
}

// toggleKind is one family of switches. Services and protocols are set through
// different routes and reported in different fields of the project, but behave
// the same way otherwise, so one implementation serves both.
type toggleKind struct {
	// typeSuffix is the resource type name after the provider prefix.
	typeSuffix  string
	noun        string
	description string
	items       []toggleItem
	update      func(client *sdkproject.Project, id string, enabled bool) (*models.Project, error)
	current     func(project *models.Project) map[string]bool
}

// toggleValues maps an item ID to its value in a plan or state. Null means the
// item is not managed.
type toggleValues map[string]types.Bool

// toggleChange is one call to make against the server.
type toggleChange struct {
	id      string
	enabled bool
}

var (
	_ resource.Resource                = &toggleResource{}
	_ resource.ResourceWithConfigure   = &toggleResource{}
	_ resource.ResourceWithImportState = &toggleResource{}
)

type toggleResource struct {
	kind    toggleKind
	clients *common.AppwriteClients
}

func (r *toggleResource) typeName() string {
	return "appwrite" + r.kind.typeSuffix
}

func (r *toggleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.kind.typeSuffix
}

func (r *toggleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description:   "The project ID.",
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"project_id": common.ProjectIDAttribute(),
	}
	for _, item := range r.kind.items {
		description := fmt.Sprintf("Whether %s is enabled.", item.name)
		if item.note != "" {
			description += " " + item.note
		}
		description += fmt.Sprintf(" Left unset, the %s is not managed; removing the attribute from the "+
			"configuration enables it again.", r.kind.noun)
		attributes[item.id] = schema.BoolAttribute{
			Description: description,
			Optional:    true,
		}
	}
	resp.Schema = schema.Schema{
		Description: r.kind.description,
		Attributes:  attributes,
	}
}

func (r *toggleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *toggleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var projectIDValue types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("project_id"), &projectIDValue)...)
	planned, diags := r.values(ctx, req.Plan.GetAttribute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, projectIDValue)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	client, err := r.client(projectID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	prior := toggleValues{}
	changes := changesFor(r.kind.items, prior, planned)
	applied, err := applyChanges(client, r.kind, changes)

	// Whatever was applied is recorded even on failure, so that the next run
	// knows which switches this resource has already changed.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), projectID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectID)...)
	r.setValues(ctx, &resp.State, resultValues(r.kind.items, prior, planned, changes[applied:]), &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error updating project %ss", r.kind.noun), r.formatError(err))
	}
}

func (r *toggleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var projectIDValue types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("project_id"), &projectIDValue)...)
	state, diags := r.values(ctx, req.State.GetAttribute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, projectIDValue)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	client, err := r.client(projectID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	project, err := client.Get()
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("Error reading project %ss", r.kind.noun), r.formatError(err))
		return
	}

	// Only managed switches are refreshed. One left unset in configuration
	// stays null, so a change made elsewhere to a switch Terraform does not
	// manage is not reported as drift.
	refreshed := refreshValues(r.kind.items, state, r.kind.current(project))
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), projectID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectID)...)
	r.setValues(ctx, &resp.State, refreshed, &resp.Diagnostics)
}

func (r *toggleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var projectIDValue types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("project_id"), &projectIDValue)...)
	prior, diags := r.values(ctx, req.State.GetAttribute)
	resp.Diagnostics.Append(diags...)
	planned, diags := r.values(ctx, req.Plan.GetAttribute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, projectIDValue)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	client, err := r.client(projectID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	changes := changesFor(r.kind.items, prior, planned)
	applied, err := applyChanges(client, r.kind, changes)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), projectID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectID)...)
	r.setValues(ctx, &resp.State, resultValues(r.kind.items, prior, planned, changes[applied:]), &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error updating project %ss", r.kind.noun), r.formatError(err))
	}
}

// Delete enables every switch this resource manages, which is Appwrite's
// default. A switch the configuration never set is left as it is.
func (r *toggleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var projectIDValue types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("project_id"), &projectIDValue)...)
	prior, diags := r.values(ctx, req.State.GetAttribute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, projectIDValue)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	client, err := r.client(projectID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	_, err = applyChanges(client, r.kind, changesFor(r.kind.items, prior, toggleValues{}))
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("Error enabling project %ss", r.kind.noun), r.formatError(err))
	}
}

// ImportState records every switch the server reports, so that the state
// mirrors the project and a configuration generated from it plans no changes.
// A switch the configuration then leaves out is planned to be enabled again.
func (r *toggleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectID := req.ID
	if projectID == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Expected project_id")
		return
	}
	client, err := r.client(projectID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}
	project, err := client.Get()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error reading project %ss", r.kind.noun), r.formatError(err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), projectID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectID)...)
	current := r.kind.current(project)
	imported := toggleValues{}
	for _, item := range r.kind.items {
		if enabled, ok := current[item.id]; ok {
			imported[item.id] = types.BoolValue(enabled)
		}
	}
	r.setValues(ctx, &resp.State, imported, &resp.Diagnostics)
}

func (r *toggleResource) client(projectID string) (*sdkproject.Project, error) {
	if err := common.ValidateProjectCredential(r.clients, r.typeName(), "project.read", "project.write"); err != nil {
		return nil, err
	}
	return appwrite.NewProject(r.clients.ClientForProject(projectID)), nil
}

func (r *toggleResource) formatError(err error) string {
	return common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance(r.typeName(), "project.read", "project.write"))
}

type attributeGetter func(context.Context, path.Path, interface{}) diag.Diagnostics

func (r *toggleResource) values(ctx context.Context, get attributeGetter) (toggleValues, diag.Diagnostics) {
	var diags diag.Diagnostics
	values := toggleValues{}
	for _, item := range r.kind.items {
		var value types.Bool
		diags.Append(get(ctx, path.Root(item.id), &value)...)
		values[item.id] = value
	}
	return values, diags
}

type attributeSetter interface {
	SetAttribute(context.Context, path.Path, interface{}) diag.Diagnostics
}

func (r *toggleResource) setValues(ctx context.Context, state attributeSetter, values toggleValues, diags *diag.Diagnostics) {
	for _, item := range r.kind.items {
		value, ok := values[item.id]
		if !ok {
			value = types.BoolNull()
		}
		diags.Append(state.SetAttribute(ctx, path.Root(item.id), value)...)
	}
}

func isManaged(value types.Bool) bool {
	return !value.IsNull() && !value.IsUnknown()
}

// changesFor returns the calls that take the server from prior to planned, in
// schema order. A switch set in planned is sent when prior does not already
// hold that value. A switch that is set in prior and not in planned is leaving
// Terraform's management, and is enabled, because that is what Appwrite does
// with a switch nobody has set.
func changesFor(items []toggleItem, prior, planned toggleValues) []toggleChange {
	var changes []toggleChange
	for _, item := range items {
		before, after := prior[item.id], planned[item.id]
		switch {
		case isManaged(after):
			if !isManaged(before) || before.ValueBool() != after.ValueBool() {
				changes = append(changes, toggleChange{id: item.id, enabled: after.ValueBool()})
			}
		case isManaged(before):
			changes = append(changes, toggleChange{id: item.id, enabled: true})
		}
	}
	return changes
}

// applyChanges makes the calls in order and stops at the first failure. It
// returns how many succeeded, so the caller can record exactly those.
func applyChanges(client *sdkproject.Project, kind toggleKind, changes []toggleChange) (int, error) {
	for i, change := range changes {
		if _, err := kind.update(client, change.id, change.enabled); err != nil {
			verb := "disabling"
			if change.enabled {
				verb = "enabling"
			}
			return i, fmt.Errorf("%s %s %q: %w", verb, kind.noun, change.id, err)
		}
	}
	return len(changes), nil
}

// resultValues is the state after an apply: the planned values, except for the
// switches in pending, whose calls were never made and which keep their prior
// value.
func resultValues(items []toggleItem, prior, planned toggleValues, pending []toggleChange) toggleValues {
	result := toggleValues{}
	for _, item := range items {
		result[item.id] = planned[item.id]
	}
	for _, change := range pending {
		result[change.id] = prior[change.id]
	}
	return result
}

// refreshValues updates the managed switches from what the server reports. A
// switch the server does not report keeps its state value: an older server
// that does not know it cannot have changed it.
func refreshValues(items []toggleItem, state toggleValues, current map[string]bool) toggleValues {
	refreshed := toggleValues{}
	for _, item := range items {
		value := state[item.id]
		if isManaged(value) {
			if enabled, ok := current[item.id]; ok {
				value = types.BoolValue(enabled)
			}
		} else {
			value = types.BoolNull()
		}
		refreshed[item.id] = value
	}
	return refreshed
}
