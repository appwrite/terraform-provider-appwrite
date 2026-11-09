package project

import (
	"context"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/id"
	"github.com/appwrite/sdk-for-go/v7/models"
	sdkproject "github.com/appwrite/sdk-for-go/v7/project"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                     = &variableResource{}
	_ resource.ResourceWithConfigure        = &variableResource{}
	_ resource.ResourceWithConfigValidators = &variableResource{}
	_ resource.ResourceWithImportState      = &variableResource{}
)

const variableResourceName = "appwrite_project_variable"

// Project variables sit behind the project scopes, not the functions or sites
// ones, even though only functions and sites read them.
var variableScopes = []string{scopeProjectRead, scopeProjectWrite}

type variableResource struct {
	clients *common.AppwriteClients
}

type variableResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Key            types.String `tfsdk:"key"`
	Value          types.String `tfsdk:"value"`
	ValueWO        types.String `tfsdk:"value_wo"`
	ValueWOVersion types.Int64  `tfsdk:"value_wo_version"`
	Secret         types.Bool   `tfsdk:"secret"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	ProjectID      types.String `tfsdk:"project_id"`
}

func NewVariableResource() resource.Resource {
	return &variableResource{}
}

func (r *variableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_variable"
}

func (r *variableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite project environment variable, shared by every function and site in the project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The variable ID.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"key": schema.StringAttribute{
				Description: "The variable key (name). Must be unique within the project. A function or site variable " +
					"with the same key takes precedence over it.",
				Required:   true,
				Validators: common.VariableKeyValidators(),
			},
			"value": schema.StringAttribute{
				Description: "The variable value. Stored in Terraform state; prefer value_wo, which is not. " +
					"Exactly one of value or value_wo must be set.",
				Optional:  true,
				Sensitive: true,
			},
			"value_wo": schema.StringAttribute{
				Description: "The variable value, as a write-only argument. Read from the configuration during " +
					"apply and never persisted. Change value_wo_version to apply a new value, since Terraform " +
					"cannot detect a change in a value it does not store. Requires Terraform 1.11 or later.",
				Optional:  true,
				Sensitive: true,
				WriteOnly: true,
				// Without a version, switching an existing resource from value
				// to value_wo leaves nothing that compares unequal, and the
				// apply reports success while the old value stays in place.
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("value_wo_version")),
				},
			},
			"value_wo_version": schema.Int64Attribute{
				Description: "Increment to apply a changed value_wo.",
				Optional:    true,
			},
			"secret": schema.BoolAttribute{
				Description: "Whether the variable is secret. Secret variables can only be updated or deleted, never " +
					"read. Appwrite makes a variable secret unless this is set to false. A secret variable cannot be " +
					"made non-secret again, so changing this from true to false forces a new resource to be created.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
					boolplanmodifier.RequiresReplaceIf(
						secretRequiresReplace,
						"Appwrite cannot make a secret variable non-secret, so the variable is recreated.",
						"Appwrite cannot make a secret variable non-secret, so the variable is recreated.",
					),
				},
			},
			"created_at": schema.StringAttribute{
				Description:   "The variable creation timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Description: "The variable last update timestamp in ISO 8601 format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					common.UseStateForUnknownUnlessUpdating(),
				},
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

// secretRequiresReplace replaces the variable when the plan unsets secret on
// one that is secret on the server. Appwrite rejects that update with
// variable_cannot_unset_secret, so an in-place plan could only fail at apply.
func secretRequiresReplace(_ context.Context, req planmodifier.BoolRequest, resp *boolplanmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = req.StateValue.ValueBool() &&
		!req.PlanValue.IsUnknown() && !req.PlanValue.IsNull() && !req.PlanValue.ValueBool()
}

// ConfigValidators enforces that a value arrives exactly one way. Making
// `value` optional to admit `value_wo` would otherwise allow a variable with no
// value at all, which the API rejects with a less obvious message.
func (r *variableResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("value"),
			path.MatchRoot("value_wo"),
		),
	}
}

func (r *variableResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *variableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan variableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	projectClient, err := r.client(projectID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	valueWO := common.WriteOnlyValue(ctx, req.Config, "value_wo", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	variable, err := createVariable(projectClient, plan, common.ResolveSecret(valueWO, plan.Value))
	if err != nil {
		resp.Diagnostics.AddError("Error creating project variable", formatVariableError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	mapVariableToState(variable, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *variableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state variableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	projectClient, err := r.client(projectID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	variable, err := projectClient.GetVariable(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading project variable", formatVariableError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	mapVariableToState(variable, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *variableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan variableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	projectClient, err := r.client(projectID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	valueWO := common.WriteOnlyValue(ctx, req.Config, "value_wo", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	variable, err := updateVariable(projectClient, plan, common.ResolveSecret(valueWO, plan.Value))
	if err != nil {
		resp.Diagnostics.AddError("Error updating project variable", formatVariableError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	mapVariableToState(variable, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *variableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state variableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	projectClient, err := r.client(projectID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	_, err = projectClient.DeleteVariable(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting project variable", formatVariableError(err))
	}
}

func (r *variableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectID, variableID, err := parseVariableImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	if projectID != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectID)...)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), variableID)...)
}

func (r *variableResource) client(projectID string) (*sdkproject.Project, error) {
	if err := common.ValidateProjectCredential(r.clients, variableResourceName, variableScopes...); err != nil {
		return nil, err
	}
	return appwrite.NewProject(r.clients.ClientForProject(projectID)), nil
}

// parseVariableImportID accepts variable_id, or project_id/variable_id for a
// variable outside the provider's default project. The project ID is empty in
// the first form.
func parseVariableImportID(importID string) (projectID, variableID string, err error) {
	parts := strings.Split(importID, "/")
	switch {
	case len(parts) == 1 && parts[0] != "":
		return "", parts[0], nil
	case len(parts) == 2 && parts[0] != "" && parts[1] != "":
		return parts[0], parts[1], nil
	default:
		return "", "", fmt.Errorf("expected variable_id or project_id/variable_id, got: %q", importID)
	}
}

func createVariable(projectClient *sdkproject.Project, plan variableResourceModel, value string) (*models.Variable, error) {
	var opts []sdkproject.CreateVariableOption
	if !plan.Secret.IsNull() && !plan.Secret.IsUnknown() {
		opts = append(opts, projectClient.WithCreateVariableSecret(plan.Secret.ValueBool()))
	}
	return projectClient.CreateVariable(id.Unique(), plan.Key.ValueString(), value, opts...)
}

// updateVariable always sends the key and value: the value may live only in
// value_wo, where Terraform cannot tell whether it changed, and resending an
// unchanged key is harmless.
func updateVariable(projectClient *sdkproject.Project, plan variableResourceModel, value string) (*models.Variable, error) {
	opts := []sdkproject.UpdateVariableOption{
		projectClient.WithUpdateVariableKey(plan.Key.ValueString()),
		projectClient.WithUpdateVariableValue(value),
	}
	if !plan.Secret.IsNull() && !plan.Secret.IsUnknown() {
		opts = append(opts, projectClient.WithUpdateVariableSecret(plan.Secret.ValueBool()))
	}
	return projectClient.UpdateVariable(plan.ID.ValueString(), opts...)
}

func mapVariableToState(variable *models.Variable, model *variableResourceModel) {
	model.ID = types.StringValue(variable.Id)
	model.Key = types.StringValue(variable.Key)
	model.Secret = types.BoolValue(variable.Secret)
	model.CreatedAt = types.StringValue(variable.CreatedAt)
	model.UpdatedAt = types.StringValue(variable.UpdatedAt)
	// A secret variable comes back with an empty value, so its value is kept
	// as configured and a change made outside Terraform goes unseen. A
	// non-secret one comes back in full, but is copied only when the
	// configuration owns `value`: copying it unconditionally would write a
	// value_wo secret into state, and on import would fill in a value the
	// configuration may supply through value_wo instead.
	if !variable.Secret && !model.Value.IsNull() {
		model.Value = types.StringValue(variable.Value)
	}
}

func formatVariableError(err error) string {
	return common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance(variableResourceName, variableScopes...))
}
