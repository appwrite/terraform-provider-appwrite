package user

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/users"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

const (
	targetResourceName = "appwrite_auth_user_target"

	providerTypeEmail = "email"
	providerTypeSMS   = "sms"
	providerTypePush  = "push"

	// Appwrite stores a target's identifier in a 255-character column and its
	// name in a 128-character one, and rejects longer values.
	maxTargetIdentifierLength = 255
	maxTargetNameLength       = 128
	// IDs are limited by the database adapter; every SQL adapter Appwrite
	// ships allows 36 characters.
	maxTargetIDLength = 36
)

var (
	// targetIDPattern mirrors Appwrite's ID rule: letters, digits, period,
	// hyphen and underscore, not starting with one of the special characters.
	targetIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

	// e164Pattern is the format Appwrite requires of an SMS target. The server
	// additionally checks the number against its table of calling codes,
	// which is data rather than syntax, so that part is left to it.
	e164Pattern = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)
)

func targetCredentialGuidance() string {
	return common.ProjectCredentialGuidance(targetResourceName, "users.read", "users.write")
}

// validateTargetIdentifier reports an identifier Appwrite would reject for the
// target's provider type.
//
// Only rules that hold for every server are checked. For email, Appwrite runs
// PHP's address validation, which is not worth reimplementing; what is checked
// here is what its parser requires before it gets that far, exactly one @ with
// something on either side, so nothing it accepts is refused at plan time.
func validateTargetIdentifier(providerType, identifier string) diag.Diagnostics {
	var diags diag.Diagnostics
	switch providerType {
	case providerTypeEmail:
		parts := strings.Split(strings.TrimSpace(identifier), "@")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			diags.AddAttributeError(path.Root("identifier"), "Invalid target identifier",
				fmt.Sprintf("email targets take an email address, got %q.", identifier))
		}
	case providerTypeSMS:
		if !e164Pattern.MatchString(identifier) {
			diags.AddAttributeError(path.Root("identifier"), "Invalid target identifier",
				fmt.Sprintf("sms targets take a phone number in E.164 format: a leading +, a country code "+
					"and at most 15 digits in all, such as +14155550100. Got %q.", identifier))
		}
	}
	return diags
}

// isContactTarget reports whether the target is the one Appwrite keeps for the
// user's own email address or phone number.
//
// Appwrite creates those itself when a user gets an email address or phone
// number, rewrites their identifier when the address or number changes, and
// deletes them when it is removed. Identifiers are unique within a project, and
// changing a user's email or phone to one a target already holds is refused,
// so a target whose identifier is the user's current address or number can
// only be that one.
func isContactTarget(user *models.User, target *models.Target) bool {
	switch target.ProviderType {
	case providerTypeEmail:
		return user.Email != "" && target.Identifier == user.Email
	case providerTypeSMS:
		return user.Phone != "" && target.Identifier == user.Phone
	default:
		return false
	}
}

// findTarget returns the user's target with the given ID, or nil.
func findTarget(user *models.User, targetID string) *models.Target {
	for i := range user.Targets {
		if user.Targets[i].Id == targetID {
			return &user.Targets[i]
		}
	}
	return nil
}

// createTarget creates the planned target.
//
// Appwrite's create route looks the provider up but never checks it: an ID
// that matches no provider is silently stored as no provider, and a provider
// of another type is accepted. Its update route checks both. So when a provider
// is configured, the create is followed by an update setting the same provider,
// and the server's own check decides. The returned target is the created one
// even when that check fails, so the caller can record what now exists.
func createTarget(client *users.Users, plan targetResourceModel, targetID string) (*models.Target, error) {
	var opts []users.CreateTargetOption
	if !plan.Name.IsNull() {
		opts = append(opts, client.WithCreateTargetName(plan.Name.ValueString()))
	}
	if !plan.ProviderID.IsNull() {
		opts = append(opts, client.WithCreateTargetProviderId(plan.ProviderID.ValueString()))
	}

	target, err := client.CreateTarget(
		plan.UserID.ValueString(),
		targetID,
		plan.ProviderType.ValueString(),
		plan.Identifier.ValueString(),
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("creating target: %w", err)
	}

	if plan.ProviderID.IsNull() {
		return target, nil
	}

	checked, err := client.UpdateTarget(target.UserId, target.Id, client.WithUpdateTargetProviderId(plan.ProviderID.ValueString()))
	if err != nil {
		return target, fmt.Errorf("setting provider %q: %w", plan.ProviderID.ValueString(), err)
	}
	return checked, nil
}

// targetUpdateOptions returns the fields that differ between state and plan.
//
// Only changed fields are sent. Sending an unchanged identifier is not a
// no-op: Appwrite clears the target's expired flag whenever it receives one.
// A field removed from the configuration never reaches here, because the API
// cannot clear it and the schema plans a replacement instead.
func targetUpdateOptions(client *users.Users, state, plan targetResourceModel) []users.UpdateTargetOption {
	var opts []users.UpdateTargetOption
	if !plan.Identifier.Equal(state.Identifier) {
		opts = append(opts, client.WithUpdateTargetIdentifier(plan.Identifier.ValueString()))
	}
	if !plan.ProviderID.IsNull() && !plan.ProviderID.Equal(state.ProviderID) {
		opts = append(opts, client.WithUpdateTargetProviderId(plan.ProviderID.ValueString()))
	}
	if !plan.Name.IsNull() && !plan.Name.Equal(state.Name) {
		opts = append(opts, client.WithUpdateTargetName(plan.Name.ValueString()))
	}
	return opts
}

// mapTargetToState copies a target into the model. Appwrite reports an unset
// name as "" and an unset provider as null; both are null in state, so that
// an omitted argument matches what the server returns.
func mapTargetToState(target *models.Target, model *targetResourceModel) {
	model.ID = types.StringValue(target.Id)
	model.UserID = types.StringValue(target.UserId)
	model.ProviderType = types.StringValue(target.ProviderType)
	model.Identifier = types.StringValue(target.Identifier)
	model.ProviderID = nullIfEmpty(common.Deref(target.ProviderId))
	model.Name = nullIfEmpty(target.Name)
	model.Expired = types.BoolValue(target.Expired)
	model.CreatedAt = types.StringValue(target.CreatedAt)
	model.UpdatedAt = types.StringValue(target.UpdatedAt)
}

func nullIfEmpty(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// formatTargetError adds what Appwrite's error leaves out.
//
// identifierChanged says whether the request set an identifier.
func formatTargetError(err error, identifierChanged bool) string {
	msg := common.FormatErrorWithAuthGuidance(err, targetCredentialGuidance())
	switch common.ErrorType(err) {
	case "user_target_already_exists":
		// The message names the ID, but a duplicate identifier gets the same
		// error, and is the likelier cause.
		return msg + "\n\nAppwrite returns this when the target ID is taken and also when another target in " +
			"the project already has this identifier, including the target Appwrite keeps for a user's own " +
			"email address or phone number. Identifiers are unique across the whole project."
	case "general_unknown":
		if identifierChanged {
			// Appwrite 2.2.0 does not catch the unique-index violation on
			// update, so a duplicate identifier comes back as a 500.
			return msg + "\n\nIf the identifier is already used by another target in the project, Appwrite " +
				"reports it as this server error. Identifiers are unique across the whole project."
		}
	}
	return msg
}
