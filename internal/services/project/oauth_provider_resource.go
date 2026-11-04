package project

import (
	"context"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/client"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &oauthProviderResource{}
	_ resource.ResourceWithConfigure      = &oauthProviderResource{}
	_ resource.ResourceWithImportState    = &oauthProviderResource{}
	_ resource.ResourceWithValidateConfig = &oauthProviderResource{}
)

const oauthProviderResourceName = "appwrite_project_oauth_provider"

var oauthProviderScopes = []string{"project.oauth2.read", "project.oauth2.write"}

type oauthProviderResource struct {
	clients *common.AppwriteClients
}

func NewOAuthProviderResource() resource.Resource {
	return &oauthProviderResource{}
}

func (r *oauthProviderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_oauth_provider"
}

func (r *oauthProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description:   "The provider ID, the same as provider_id.",
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"provider_id": schema.StringAttribute{
			Description: "The OAuth2 provider to configure, for example github, google or oidc. The resource " +
				"documentation lists every provider and the arguments it takes. " + common.ForcesReplacementNote,
			Required:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			Validators:    []validator.String{stringvalidator.OneOf(oauthProviderIDs()...)},
		},
		"enabled": schema.BoolAttribute{
			Description: "Whether users can sign in with the provider. Defaults to true. Enabling some providers " +
				"makes Appwrite check the client ID and secret with the provider, and the apply fails if they " +
				"are rejected.",
			Optional: true,
			Computed: true,
			Default:  booldefault.StaticBool(true),
		},
		"client_id": schema.StringAttribute{
			Description: "The OAuth2 application's client ID, under whatever name the provider gives it: app " +
				"ID, application ID, app key, or for Apple the Services ID.",
			Optional: true,
		},
		"client_secret": schema.StringAttribute{
			Description: "The OAuth2 application's client secret, or for Apple the contents of the .p8 key file. " +
				"Appwrite never returns it, so a change made outside Terraform is not detected. Stored in " +
				"Terraform state; prefer client_secret_wo, which is not.",
			Optional:   true,
			Sensitive:  true,
			Validators: []validator.String{stringvalidator.ConflictsWith(path.MatchRoot("client_secret_wo"))},
		},
		"client_secret_wo": schema.StringAttribute{
			Description: "The client secret, as a write-only argument. Read from the configuration during apply " +
				"and never persisted. Change client_secret_wo_version to apply a new value. Requires Terraform " +
				"1.11 or later.",
			Optional:  true,
			Sensitive: true,
			WriteOnly: true,
			// Without a version, switching from client_secret to this
			// argument changes nothing Terraform compares, so the new secret
			// would never be sent.
			Validators: []validator.String{stringvalidator.AlsoRequires(path.MatchRoot("client_secret_wo_version"))},
		},
		"client_secret_wo_version": schema.Int64Attribute{
			Description: "Increment to apply a changed client_secret_wo.",
			Optional:    true,
		},
		"project_id": common.ProjectIDAttribute(),
	}
	for name, extra := range oauthExtras {
		attributes[name] = extraAttribute(name, extra)
	}

	resp.Schema = schema.Schema{
		Description: "Configures an OAuth2 sign-in provider of an Appwrite project. Every provider exists in " +
			"every project, so creating this resource sets the provider up and destroying it disables the " +
			"provider and clears its credentials.",
		Attributes: attributes,
	}
}

func extraAttribute(name string, extra oauthExtra) schema.Attribute {
	description := extraDescription(name)
	switch extra.kind {
	case extraBool:
		return schema.BoolAttribute{Description: description, Optional: true}
	case extraInt64:
		attribute := schema.Int64Attribute{Description: description, Optional: true}
		if extra.serverDefault {
			attribute.Computed = true
			attribute.PlanModifiers = []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}
		}
		return attribute
	case extraSet:
		attribute := schema.SetAttribute{Description: description, Optional: true, ElementType: types.StringType}
		if extra.serverDefault {
			attribute.Computed = true
			attribute.PlanModifiers = []planmodifier.Set{setplanmodifier.UseStateForUnknown()}
		}
		return attribute
	default:
		return schema.StringAttribute{Description: description, Optional: true}
	}
}

func (r *oauthProviderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValidateConfig rejects a provider-specific argument given to a provider that
// does not take it. Appwrite ignores parameters a route does not declare, so
// without this the setting would be silently dropped.
func (r *oauthProviderResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config oauthProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.ProviderID.IsNull() || config.ProviderID.IsUnknown() {
		return
	}
	resp.Diagnostics.Append(validateOAuthProviderConfig(&config)...)
}

func validateOAuthProviderConfig(config *oauthProviderModel) diag.Diagnostics {
	var diags diag.Diagnostics
	p, ok := lookupOAuthProvider(config.ProviderID.ValueString())
	if !ok {
		// The provider_id validator reports this one.
		return diags
	}
	for _, name := range sortedExtraNames() {
		if p.takes(name) || !isSet(config.extra(name)) {
			continue
		}
		diags.AddAttributeError(
			path.Root(name),
			"Argument not supported by this provider",
			fmt.Sprintf("%s is not an argument of the %s provider. It applies only to %s.",
				name, p.id, strings.Join(providersTaking(name), ", ")),
		)
	}
	return diags
}

// isSet reports whether a configuration gives a value. One unknown until
// apply may turn out null, so only a known one is judged.
func isSet(field any) bool {
	switch v := field.(type) {
	case *types.String:
		return !v.IsNull() && !v.IsUnknown()
	case *types.Bool:
		return !v.IsNull() && !v.IsUnknown()
	case *types.Int64:
		return !v.IsNull() && !v.IsUnknown()
	case *types.Set:
		return !v.IsNull() && !v.IsUnknown()
	}
	return false
}

func (r *oauthProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan oauthProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, req.Config, &plan, &resp.Diagnostics, "Error configuring OAuth2 provider")
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *oauthProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan oauthProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, req.Config, &plan, &resp.Diagnostics, "Error updating OAuth2 provider")
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// apply is Create and Update alike: the provider already exists, so both send
// its update route with the whole configuration.
func (r *oauthProviderResource) apply(ctx context.Context, config tfsdk.Config, plan *oauthProviderModel, diagnostics *diag.Diagnostics, summary string) {
	p, ok := lookupOAuthProvider(plan.ProviderID.ValueString())
	if !ok {
		diagnostics.AddAttributeError(path.Root("provider_id"), "Unknown OAuth2 provider", fmt.Sprintf("%q is not a provider this resource supports.", plan.ProviderID.ValueString()))
		return
	}
	projectID, c, err := r.client(plan.ProjectID)
	if err != nil {
		diagnostics.AddError("Error resolving project", err.Error())
		return
	}

	// A write-only argument never reaches the plan, so it is read from config.
	secretWO := common.WriteOnlyValue(ctx, config, "client_secret_wo", diagnostics)
	if diagnostics.HasError() {
		return
	}
	secret, hasSecret := "", false
	switch {
	case !secretWO.IsNull() && !secretWO.IsUnknown():
		secret, hasSecret = secretWO.ValueString(), true
	case !plan.ClientSecret.IsNull() && !plan.ClientSecret.IsUnknown():
		secret, hasSecret = plan.ClientSecret.ValueString(), true
	}

	body, err := updateOAuthProvider(c, p.id, updateParams(p, plan, secret, hasSecret))
	if err != nil {
		diagnostics.AddError(summary, common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance(oauthProviderResourceName, oauthProviderScopes...)))
		return
	}
	plan.ProjectID = types.StringValue(projectID)
	applyOAuthResponse(p, body, plan)
}

func (r *oauthProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state oauthProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, ok := lookupOAuthProvider(state.ProviderID.ValueString())
	if !ok {
		resp.Diagnostics.AddAttributeError(path.Root("provider_id"), "Unknown OAuth2 provider", fmt.Sprintf("%q is not a provider this resource supports.", state.ProviderID.ValueString()))
		return
	}
	projectID, c, err := r.client(state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project", err.Error())
		return
	}

	body, err := getOAuthProvider(c, p.id)
	if err != nil {
		// The provider itself cannot be missing; a 404 means the project is.
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading OAuth2 provider", common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance(oauthProviderResourceName, oauthProviderScopes...)))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	applyOAuthResponse(p, body, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete disables the provider and clears its credentials. The provider is
// part of the project and cannot be removed.
func (r *oauthProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state oauthProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, ok := lookupOAuthProvider(state.ProviderID.ValueString())
	if !ok {
		return
	}
	_, c, err := r.client(state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project", err.Error())
		return
	}

	if _, err := updateOAuthProvider(c, p.id, resetParams(p)); err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error disabling OAuth2 provider", common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance(oauthProviderResourceName, oauthProviderScopes...)))
	}
}

// ImportState accepts provider_id or project_id/provider_id.
func (r *oauthProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	const expected = "Expected provider_id or project_id/provider_id, for example github or my-project/github"
	parts := strings.Split(req.ID, "/")
	for _, part := range parts {
		if part == "" {
			resp.Diagnostics.AddError("Invalid import ID", expected)
			return
		}
	}

	var providerID string
	switch len(parts) {
	case 1:
		providerID = parts[0]
	case 2:
		providerID = parts[1]
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("%s, got: %s", expected, req.ID))
		return
	}
	if _, ok := lookupOAuthProvider(providerID); !ok {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("%q is not an OAuth2 provider this resource supports. Supported providers: %s.", providerID, strings.Join(oauthProviderIDs(), ", ")))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("provider_id"), providerID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), providerID)...)
}

func (r *oauthProviderResource) client(projectIDValue types.String) (string, client.Client, error) {
	projectID, err := common.ResolveProjectID(r.clients, projectIDValue)
	if err != nil {
		return "", client.Client{}, err
	}
	if err := common.ValidateProjectCredential(r.clients, oauthProviderResourceName, oauthProviderScopes...); err != nil {
		return "", client.Client{}, err
	}
	return projectID, r.clients.ClientForProject(projectID), nil
}
