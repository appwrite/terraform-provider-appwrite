package project

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

const authPoliciesResourceName = "appwrite_project_auth_policies"

// project.policies.* replaced policies.*, which Appwrite still accepts.
var authPoliciesScopes = []string{"project.policies.read", "project.policies.write"}

var (
	_ resource.Resource                = &authPoliciesResource{}
	_ resource.ResourceWithConfigure   = &authPoliciesResource{}
	_ resource.ResourceWithImportState = &authPoliciesResource{}
)

type authPoliciesResource struct {
	clients *common.AppwriteClients
}

type authPoliciesResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	ProjectID            types.String `tfsdk:"project_id"`
	PasswordDictionary   types.Bool   `tfsdk:"password_dictionary"`
	PasswordHistory      types.Int64  `tfsdk:"password_history"`
	PasswordPersonalData types.Bool   `tfsdk:"password_personal_data"`
	PasswordStrength     types.Object `tfsdk:"password_strength"`
	PasswordPwned        types.Object `tfsdk:"password_pwned"`
	SessionAlert         types.Bool   `tfsdk:"session_alert"`
	SessionDuration      types.Int64  `tfsdk:"session_duration"`
	SessionInvalidation  types.Bool   `tfsdk:"session_invalidation"`
	SessionLimit         types.Int64  `tfsdk:"session_limit"`
	UserLimit            types.Int64  `tfsdk:"user_limit"`
	MembershipPrivacy    types.Object `tfsdk:"membership_privacy"`
	MFAFactors           types.Object `tfsdk:"mfa_factors"`
	DenyAliasedEmail     types.Bool   `tfsdk:"deny_aliased_email"`
	DenyDisposableEmail  types.Bool   `tfsdk:"deny_disposable_email"`
	DenyFreeEmail        types.Bool   `tfsdk:"deny_free_email"`
	DenyCorporateEmail   types.Bool   `tfsdk:"deny_corporate_email"`
}

type passwordStrengthModel struct {
	Min       types.Int64 `tfsdk:"min"`
	Uppercase types.Bool  `tfsdk:"uppercase"`
	Lowercase types.Bool  `tfsdk:"lowercase"`
	Number    types.Bool  `tfsdk:"number"`
	Symbols   types.Bool  `tfsdk:"symbols"`
}

type passwordPwnedModel struct {
	Enabled  types.Bool `tfsdk:"enabled"`
	Sessions types.Bool `tfsdk:"sessions"`
	Users    types.Bool `tfsdk:"users"`
}

type membershipPrivacyModel struct {
	UserID         types.Bool `tfsdk:"user_id"`
	UserEmail      types.Bool `tfsdk:"user_email"`
	UserPhone      types.Bool `tfsdk:"user_phone"`
	UserName       types.Bool `tfsdk:"user_name"`
	UserMFA        types.Bool `tfsdk:"user_mfa"`
	UserAccessedAt types.Bool `tfsdk:"user_accessed_at"`
}

type mfaFactorsModel struct {
	TOTP   types.Bool `tfsdk:"totp"`
	Email  types.Bool `tfsdk:"email"`
	Phone  types.Bool `tfsdk:"phone"`
	Custom types.Bool `tfsdk:"custom"`
}

var (
	passwordStrengthAttrTypes = map[string]attr.Type{
		"min": types.Int64Type, "uppercase": types.BoolType, "lowercase": types.BoolType,
		"number": types.BoolType, "symbols": types.BoolType,
	}
	passwordPwnedAttrTypes = map[string]attr.Type{
		"enabled": types.BoolType, "sessions": types.BoolType, "users": types.BoolType,
	}
	membershipPrivacyAttrTypes = map[string]attr.Type{
		"user_id": types.BoolType, "user_email": types.BoolType, "user_phone": types.BoolType,
		"user_name": types.BoolType, "user_mfa": types.BoolType, "user_accessed_at": types.BoolType,
	}
	mfaFactorsAttrTypes = map[string]attr.Type{
		"totp": types.BoolType, "email": types.BoolType, "phone": types.BoolType, "custom": types.BoolType,
	}
)

func NewAuthPoliciesResource() resource.Resource {
	return &authPoliciesResource{}
}

func (r *authPoliciesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_auth_policies"
}

func boolSetting(description string, def bool) schema.BoolAttribute {
	return schema.BoolAttribute{
		Description: fmt.Sprintf("%s Defaults to `%t`.", description, def),
		Optional:    true,
		Computed:    true,
		Default:     booldefault.StaticBool(def),
	}
}

func int64Setting(description string, def, lo, hi int64) schema.Int64Attribute {
	return schema.Int64Attribute{
		Description: fmt.Sprintf("%s Defaults to `%d`.", description, def),
		Optional:    true,
		Computed:    true,
		Default:     int64default.StaticInt64(def),
		Validators:  []validator.Int64{int64validator.Between(lo, hi)},
	}
}

// objectSetting is a policy with several fields. Leaving the whole block out
// sets every field to its default, and so does leaving a field out of it.
func objectSetting(ctx context.Context, description string, attributes map[string]schema.Attribute, attrTypes map[string]attr.Type, def any) schema.SingleNestedAttribute {
	value, diags := types.ObjectValueFrom(ctx, attrTypes, def)
	if diags.HasError() {
		panic(fmt.Sprintf("default for %s: %v", description, diags))
	}
	return schema.SingleNestedAttribute{
		Description: description + " Leaving it out sets every field to its default.",
		Optional:    true,
		Computed:    true,
		Default:     objectdefault.StaticValue(value),
		Attributes:  attributes,
	}
}

func (r *authPoliciesResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	d := defaultAuthPolicies
	resp.Schema = schema.Schema{
		Description: "Manages a project's password, session, user, team membership, MFA and sign-up email policies. " +
			singletonDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The project ID.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": common.ProjectIDAttribute(),
			"password_dictionary": boolSetting("Whether a new password is rejected when it is one of the 10,000 most "+
				"common passwords. Existing passwords stay valid.", d.PasswordDictionary),
			"password_history": int64Setting(fmt.Sprintf("How many of a user's previous passwords they cannot reuse, "+
				"from 1 to %d, or `0` to keep no history. Passwords are only recorded while the history is kept, so "+
				"turning it on does not cover passwords set before.", maxPasswordHistory),
				d.PasswordHistory, 0, maxPasswordHistory),
			"password_personal_data": boolSetting("Whether a new password is rejected when it contains the user's ID, "+
				"name, email address or phone number. Existing passwords stay valid.", d.PasswordPersonalData),
			"password_strength": objectSetting(ctx, "What a new password must contain.",
				map[string]schema.Attribute{
					"min": int64Setting(fmt.Sprintf("The minimum length, from %d to %d.", minPasswordStrength, maxPasswordStrength),
						d.PasswordStrength.Min, minPasswordStrength, maxPasswordStrength),
					"uppercase": boolSetting("Whether it must contain an uppercase letter.", d.PasswordStrength.Uppercase),
					"lowercase": boolSetting("Whether it must contain a lowercase letter.", d.PasswordStrength.Lowercase),
					"number":    boolSetting("Whether it must contain a digit.", d.PasswordStrength.Number),
					"symbols":   boolSetting("Whether it must contain a symbol.", d.PasswordStrength.Symbols),
				}, passwordStrengthAttrTypes, passwordStrengthToModel(d.PasswordStrength)),
			"password_pwned": objectSetting(ctx, "Checks passwords against the Have I Been Pwned breach database. "+
				"Only the first five characters of the password's hash are sent to it. Requires Appwrite 2.3.0 or "+
				"later; against an older server, any value other than the defaults fails to apply.",
				map[string]schema.Attribute{
					"enabled": boolSetting("Whether passwords are checked when a user signs up, signs in or sets a "+
						"password. On its own this only records the result on the user.", d.PasswordPwned.Enabled),
					"users": boolSetting("Whether a breached password is rejected when a user signs up or sets "+
						"a new password.", d.PasswordPwned.Users),
					"sessions": boolSetting("Whether signing in with a breached password is refused until the "+
						"password is reset.", d.PasswordPwned.Sessions),
				}, passwordPwnedAttrTypes, passwordPwnedToModel(d.PasswordPwned)),
			"session_alert": boolSetting("Whether a user is emailed when a new session is created for them. The "+
				"first session after signing up does not send one.", d.SessionAlert),
			"session_duration": int64Setting(fmt.Sprintf("How long a session lasts, in seconds, from %d to %d "+
				"(one year).", minSessionDuration, maxSessionDuration),
				d.SessionDuration, minSessionDuration, maxSessionDuration),
			"session_invalidation": boolSetting("Whether changing a user's password ends their existing sessions.",
				d.SessionInvalidation),
			"session_limit": int64Setting(fmt.Sprintf("The most sessions a user can have at once, from %d to %d. "+
				"Creating one more deletes the oldest.", minSessionLimit, maxSessionLimit),
				d.SessionLimit, minSessionLimit, maxSessionLimit),
			"user_limit": int64Setting(fmt.Sprintf("The most users the project can have, from 1 to %d, or `0` for "+
				"no limit. Reaching it stops new sign-ups; existing users are not affected.", maxUserLimit),
				d.UserLimit, 0, maxUserLimit),
			"membership_privacy": objectSetting(ctx, "Which details of a team member the other members of the "+
				"team can see.",
				map[string]schema.Attribute{
					"user_id":          boolSetting("Whether members can see each other's user ID.", d.MembershipPrivacy.UserID),
					"user_email":       boolSetting("Whether members can see each other's email address.", d.MembershipPrivacy.UserEmail),
					"user_phone":       boolSetting("Whether members can see each other's phone number.", d.MembershipPrivacy.UserPhone),
					"user_name":        boolSetting("Whether members can see each other's name.", d.MembershipPrivacy.UserName),
					"user_mfa":         boolSetting("Whether members can see if other members have MFA enabled.", d.MembershipPrivacy.UserMFA),
					"user_accessed_at": boolSetting("Whether members can see when other members last accessed the project.", d.MembershipPrivacy.UserAccessedAt),
				}, membershipPrivacyAttrTypes, membershipPrivacyToModel(d.MembershipPrivacy)),
			"mfa_factors": objectSetting(ctx, "Which factors users can complete an MFA challenge with. Recovery "+
				"codes always remain available.",
				map[string]schema.Attribute{
					"totp":   boolSetting("Whether an authenticator app (TOTP) can be used.", d.MFAFactors.TOTP),
					"email":  boolSetting("Whether a code sent by email can be used.", d.MFAFactors.Email),
					"phone":  boolSetting("Whether a code sent by SMS can be used.", d.MFAFactors.Phone),
					"custom": boolSetting("Whether a code delivered through your own channel can be used.", d.MFAFactors.Custom),
				}, mfaFactorsAttrTypes, mfaFactorsToModel(d.MFAFactors)),
			"deny_aliased_email": boolSetting("Whether signing up or changing email address is refused for an "+
				"aliased address, such as one with a `+` suffix.", d.DenyAliasedEmail),
			"deny_disposable_email": boolSetting("Whether signing up or changing email address is refused for an "+
				"address at a known disposable email domain.", d.DenyDisposableEmail),
			"deny_free_email": boolSetting("Whether signing up or changing email address is refused for an "+
				"address at a free email provider such as Gmail or Yahoo.", d.DenyFreeEmail),
			"deny_corporate_email": boolSetting("Whether signing up or changing email address is restricted to "+
				"corporate addresses, refusing addresses at free or disposable email domains. Despite the name, "+
				"which follows Appwrite's policy ID, it is non-corporate addresses that are denied.", d.DenyCorporateEmail),
		},
	}
}

func (r *authPoliciesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *authPoliciesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan authPoliciesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &plan, &resp.State, &resp.Diagnostics)
}

func (r *authPoliciesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan authPoliciesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &plan, &resp.State, &resp.Diagnostics)
}

func (r *authPoliciesResource) apply(ctx context.Context, plan *authPoliciesResourceModel, state *tfsdk.State, diags *diag.Diagnostics) {
	projectID, client, ok := r.client(plan.ProjectID, diags)
	if !ok {
		return
	}
	want, d := policiesFromModel(ctx, plan)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	unlock := lockAuthSettings(projectID)
	result, err := applyAuthPolicies(client, want)
	unlock()
	if err != nil {
		if errors.Is(err, errPolicyUnsupported) {
			diags.AddError("Unsupported auth policy", err.Error())
			return
		}
		diags.AddError("Error updating auth policies", common.FormatErrorWithAuthGuidance(err, authPoliciesGuidance()))
		return
	}

	plan.ID = types.StringValue(projectID)
	plan.ProjectID = types.StringValue(projectID)
	diags.Append(policiesToModel(ctx, result.policies, plan)...)
	diags.Append(state.Set(ctx, plan)...)
}

func (r *authPoliciesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state authPoliciesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectID, client, ok := r.client(state.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	result, err := client.list()
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading auth policies", common.FormatErrorWithAuthGuidance(err, authPoliciesGuidance()))
		return
	}

	state.ID = types.StringValue(projectID)
	state.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(policiesToModel(ctx, result.policies, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete puts every policy back to the value a new project has.
func (r *authPoliciesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state authPoliciesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectID, client, ok := r.client(state.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	unlock := lockAuthSettings(projectID)
	_, err := applyAuthPolicies(client, defaultAuthPolicies)
	unlock()
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error resetting auth policies", common.FormatErrorWithAuthGuidance(err, authPoliciesGuidance()))
	}
}

func (r *authPoliciesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importProjectSettings(ctx, req, resp)
}

func (r *authPoliciesResource) client(configured types.String, diags *diag.Diagnostics) (string, *authPoliciesClient, bool) {
	projectID, err := common.ResolveProjectID(r.clients, configured)
	if err != nil {
		diags.AddError("Missing project_id", err.Error())
		return "", nil, false
	}
	if err := common.ValidateProjectCredential(r.clients, authPoliciesResourceName, authPoliciesScopes...); err != nil {
		diags.AddError("Unsupported Appwrite credential", err.Error())
		return "", nil, false
	}
	return projectID, newAuthPoliciesClient(r.clients.ClientForProject(projectID)), true
}

func authPoliciesGuidance() string {
	return common.ProjectCredentialGuidance(authPoliciesResourceName, authPoliciesScopes...)
}

func policiesFromModel(ctx context.Context, m *authPoliciesResourceModel) (authPolicies, diag.Diagnostics) {
	var diags diag.Diagnostics
	var strength passwordStrengthModel
	var pwned passwordPwnedModel
	var membership membershipPrivacyModel
	var mfa mfaFactorsModel
	opts := basetypes.ObjectAsOptions{}
	diags.Append(m.PasswordStrength.As(ctx, &strength, opts)...)
	diags.Append(m.PasswordPwned.As(ctx, &pwned, opts)...)
	diags.Append(m.MembershipPrivacy.As(ctx, &membership, opts)...)
	diags.Append(m.MFAFactors.As(ctx, &mfa, opts)...)

	return authPolicies{
		PasswordDictionary:   m.PasswordDictionary.ValueBool(),
		PasswordHistory:      m.PasswordHistory.ValueInt64(),
		PasswordPersonalData: m.PasswordPersonalData.ValueBool(),
		PasswordStrength: passwordStrengthPolicy{
			Min:       strength.Min.ValueInt64(),
			Uppercase: strength.Uppercase.ValueBool(),
			Lowercase: strength.Lowercase.ValueBool(),
			Number:    strength.Number.ValueBool(),
			Symbols:   strength.Symbols.ValueBool(),
		},
		PasswordPwned: passwordPwnedPolicy{
			Enabled:  pwned.Enabled.ValueBool(),
			Sessions: pwned.Sessions.ValueBool(),
			Users:    pwned.Users.ValueBool(),
		},
		SessionAlert:        m.SessionAlert.ValueBool(),
		SessionDuration:     m.SessionDuration.ValueInt64(),
		SessionInvalidation: m.SessionInvalidation.ValueBool(),
		SessionLimit:        m.SessionLimit.ValueInt64(),
		UserLimit:           m.UserLimit.ValueInt64(),
		MembershipPrivacy: membershipPrivacyPolicy{
			UserID:         membership.UserID.ValueBool(),
			UserEmail:      membership.UserEmail.ValueBool(),
			UserPhone:      membership.UserPhone.ValueBool(),
			UserName:       membership.UserName.ValueBool(),
			UserMFA:        membership.UserMFA.ValueBool(),
			UserAccessedAt: membership.UserAccessedAt.ValueBool(),
		},
		MFAFactors: mfaFactorsPolicy{
			TOTP:   mfa.TOTP.ValueBool(),
			Email:  mfa.Email.ValueBool(),
			Phone:  mfa.Phone.ValueBool(),
			Custom: mfa.Custom.ValueBool(),
		},
		DenyAliasedEmail:    m.DenyAliasedEmail.ValueBool(),
		DenyDisposableEmail: m.DenyDisposableEmail.ValueBool(),
		DenyFreeEmail:       m.DenyFreeEmail.ValueBool(),
		DenyCorporateEmail:  m.DenyCorporateEmail.ValueBool(),
	}, diags
}

// policiesToModel copies the policies into the model. A policy the server
// does not report is read as its default, which is the only value
// applyAuthPolicies accepts for it.
func policiesToModel(ctx context.Context, p authPolicies, m *authPoliciesResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	object := func(attrTypes map[string]attr.Type, value any) types.Object {
		o, d := types.ObjectValueFrom(ctx, attrTypes, value)
		diags.Append(d...)
		return o
	}

	m.PasswordDictionary = types.BoolValue(p.PasswordDictionary)
	m.PasswordHistory = types.Int64Value(p.PasswordHistory)
	m.PasswordPersonalData = types.BoolValue(p.PasswordPersonalData)
	m.PasswordStrength = object(passwordStrengthAttrTypes, passwordStrengthToModel(p.PasswordStrength))
	m.PasswordPwned = object(passwordPwnedAttrTypes, passwordPwnedToModel(p.PasswordPwned))
	m.SessionAlert = types.BoolValue(p.SessionAlert)
	m.SessionDuration = types.Int64Value(p.SessionDuration)
	m.SessionInvalidation = types.BoolValue(p.SessionInvalidation)
	m.SessionLimit = types.Int64Value(p.SessionLimit)
	m.UserLimit = types.Int64Value(p.UserLimit)
	m.MembershipPrivacy = object(membershipPrivacyAttrTypes, membershipPrivacyToModel(p.MembershipPrivacy))
	m.MFAFactors = object(mfaFactorsAttrTypes, mfaFactorsToModel(p.MFAFactors))
	m.DenyAliasedEmail = types.BoolValue(p.DenyAliasedEmail)
	m.DenyDisposableEmail = types.BoolValue(p.DenyDisposableEmail)
	m.DenyFreeEmail = types.BoolValue(p.DenyFreeEmail)
	m.DenyCorporateEmail = types.BoolValue(p.DenyCorporateEmail)
	return diags
}

func passwordStrengthToModel(p passwordStrengthPolicy) passwordStrengthModel {
	return passwordStrengthModel{
		Min:       types.Int64Value(p.Min),
		Uppercase: types.BoolValue(p.Uppercase),
		Lowercase: types.BoolValue(p.Lowercase),
		Number:    types.BoolValue(p.Number),
		Symbols:   types.BoolValue(p.Symbols),
	}
}

func passwordPwnedToModel(p passwordPwnedPolicy) passwordPwnedModel {
	return passwordPwnedModel{
		Enabled:  types.BoolValue(p.Enabled),
		Sessions: types.BoolValue(p.Sessions),
		Users:    types.BoolValue(p.Users),
	}
}

func membershipPrivacyToModel(p membershipPrivacyPolicy) membershipPrivacyModel {
	return membershipPrivacyModel{
		UserID:         types.BoolValue(p.UserID),
		UserEmail:      types.BoolValue(p.UserEmail),
		UserPhone:      types.BoolValue(p.UserPhone),
		UserName:       types.BoolValue(p.UserName),
		UserMFA:        types.BoolValue(p.UserMFA),
		UserAccessedAt: types.BoolValue(p.UserAccessedAt),
	}
}

func mfaFactorsToModel(p mfaFactorsPolicy) mfaFactorsModel {
	return mfaFactorsModel{
		TOTP:   types.BoolValue(p.TOTP),
		Email:  types.BoolValue(p.Email),
		Phone:  types.BoolValue(p.Phone),
		Custom: types.BoolValue(p.Custom),
	}
}
