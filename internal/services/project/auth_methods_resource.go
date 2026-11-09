package project

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/models"
	sdkproject "github.com/appwrite/sdk-for-go/v7/project"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

const authMethodsResourceName = "appwrite_project_auth_methods"

// Auth methods are read from GET /project and written with
// PATCH /project/auth-methods/{methodId}, both under the project scopes.
var authMethodsScopes = []string{scopeProjectRead, scopeProjectWrite}

// authMethod pairs an attribute with the method ID Appwrite uses for it. Every
// method is enabled on a new project.
type authMethod struct {
	attribute   string
	id          string
	description string
}

var authMethods = []authMethod{
	{"email_password", "email-password", "Whether users can sign up and sign in with an email address and password."},
	{"magic_url", "magic-url", "Whether users can sign in with a magic URL sent to their email address."},
	{"email_otp", "email-otp", "Whether users can sign in with a one-time code sent to their email address."},
	{"anonymous", "anonymous", "Whether users can create anonymous sessions."},
	{"invites", "invites", "Whether users can join the project by accepting a team invitation."},
	{"jwt", "jwt", "Whether users can create JWTs for their session."},
	{"phone", "phone", "Whether users can sign in with a one-time code sent to their phone number by SMS."},
}

var (
	_ resource.Resource                = &authMethodsResource{}
	_ resource.ResourceWithConfigure   = &authMethodsResource{}
	_ resource.ResourceWithImportState = &authMethodsResource{}
)

type authMethodsResource struct {
	clients *common.AppwriteClients
}

type authMethodsResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ProjectID     types.String `tfsdk:"project_id"`
	EmailPassword types.Bool   `tfsdk:"email_password"`
	MagicURL      types.Bool   `tfsdk:"magic_url"`
	EmailOTP      types.Bool   `tfsdk:"email_otp"`
	Anonymous     types.Bool   `tfsdk:"anonymous"`
	Invites       types.Bool   `tfsdk:"invites"`
	JWT           types.Bool   `tfsdk:"jwt"`
	Phone         types.Bool   `tfsdk:"phone"`
}

// fields returns the model's method attributes keyed by method ID, so the
// rest of the resource can treat them as one table.
func (m *authMethodsResourceModel) fields() map[string]*types.Bool {
	return map[string]*types.Bool{
		"email-password": &m.EmailPassword,
		"magic-url":      &m.MagicURL,
		"email-otp":      &m.EmailOTP,
		"anonymous":      &m.Anonymous,
		"invites":        &m.Invites,
		"jwt":            &m.JWT,
		"phone":          &m.Phone,
	}
}

func NewAuthMethodsResource() resource.Resource {
	return &authMethodsResource{}
}

func (r *authMethodsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_auth_methods"
}

func (r *authMethodsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description:   projectIDDescription,
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"project_id": common.ProjectIDAttribute(),
	}
	for _, method := range authMethods {
		attributes[method.attribute] = schema.BoolAttribute{
			Description: method.description + " Defaults to `true`.",
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(true),
		}
	}

	resp.Schema = schema.Schema{
		Description: "Enables or disables the ways users can sign in to a project. Requests made with an API key are not " +
			"affected. " + singletonDescription,
		Attributes: attributes,
	}
}

func (r *authMethodsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *authMethodsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan authMethodsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &plan, &resp.State, &resp.Diagnostics)
}

func (r *authMethodsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan authMethodsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &plan, &resp.State, &resp.Diagnostics)
}

func (r *authMethodsResource) apply(ctx context.Context, plan *authMethodsResourceModel, state *tfsdk.State, diags *diag.Diagnostics) {
	projectID, client, ok := r.client(plan.ProjectID, diags)
	if !ok {
		return
	}

	desired := map[string]bool{}
	for methodID, value := range plan.fields() {
		desired[methodID] = value.ValueBool()
	}

	unlock := lockAuthSettings(projectID)
	project, err := applyAuthMethods(client, desired)
	unlock()
	if err != nil {
		diags.AddError("Error updating auth methods", common.FormatErrorWithAuthGuidance(err, authMethodsGuidance()))
		return
	}

	plan.ID = types.StringValue(projectID)
	plan.ProjectID = types.StringValue(projectID)
	mapAuthMethodsToState(project, plan)
	diags.Append(state.Set(ctx, plan)...)
}

func (r *authMethodsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state authMethodsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectID, client, ok := r.client(state.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	project, err := client.Get()
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading auth methods", common.FormatErrorWithAuthGuidance(err, authMethodsGuidance()))
		return
	}

	state.ID = types.StringValue(projectID)
	state.ProjectID = types.StringValue(projectID)
	mapAuthMethodsToState(project, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete enables every method again, which is how a new project starts.
func (r *authMethodsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state authMethodsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectID, client, ok := r.client(state.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	unlock := lockAuthSettings(projectID)
	_, err := applyAuthMethods(client, defaultAuthMethods())
	unlock()
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error resetting auth methods", common.FormatErrorWithAuthGuidance(err, authMethodsGuidance()))
	}
}

func (r *authMethodsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importProjectSettings(ctx, req, resp)
}

func (r *authMethodsResource) client(configured types.String, diags *diag.Diagnostics) (string, *sdkproject.Project, bool) {
	projectID, err := common.ResolveProjectID(r.clients, configured)
	if err != nil {
		diags.AddError("Missing project_id", err.Error())
		return "", nil, false
	}
	if err := common.ValidateProjectCredential(r.clients, authMethodsResourceName, authMethodsScopes...); err != nil {
		diags.AddError("Unsupported Appwrite credential", err.Error())
		return "", nil, false
	}
	return projectID, appwrite.NewProject(r.clients.ClientForProject(projectID)), true
}

func authMethodsGuidance() string {
	return common.ProjectCredentialGuidance(authMethodsResourceName, authMethodsScopes...)
}

func defaultAuthMethods() map[string]bool {
	defaults := make(map[string]bool, len(authMethods))
	for _, method := range authMethods {
		defaults[method.id] = true
	}
	return defaults
}

// applyAuthMethods changes the methods whose current state differs from the
// one wanted, and returns the project as the last request left it. Methods
// already in the wanted state are not sent, so an apply that changes one
// method makes one write.
func applyAuthMethods(client *sdkproject.Project, desired map[string]bool) (*models.Project, error) {
	project, err := client.Get()
	if err != nil {
		return nil, err
	}
	current := authMethodStates(project)

	for _, method := range authMethods {
		want, managed := desired[method.id]
		if !managed {
			continue
		}
		if have, reported := current[method.id]; reported && have == want {
			continue
		}
		project, err = client.UpdateAuthMethod(method.id, want)
		if err != nil {
			return nil, fmt.Errorf("auth method %s: %w", method.id, err)
		}
	}
	return project, nil
}

func authMethodStates(project *models.Project) map[string]bool {
	states := make(map[string]bool, len(project.AuthMethods))
	for _, method := range project.AuthMethods {
		states[method.Id] = method.Enabled
	}
	return states
}

// mapAuthMethodsToState copies the methods the server reports into the model.
// A method the server does not report keeps the value already in the model,
// or its default when there is none, as after an import.
func mapAuthMethodsToState(project *models.Project, model *authMethodsResourceModel) {
	states := authMethodStates(project)
	for methodID, field := range model.fields() {
		if enabled, ok := states[methodID]; ok {
			*field = types.BoolValue(enabled)
		} else if field.IsNull() || field.IsUnknown() {
			*field = types.BoolValue(true)
		}
	}
}
