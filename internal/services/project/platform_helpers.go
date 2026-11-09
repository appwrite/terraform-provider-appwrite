package project

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/appwrite/sdk-for-go/v7/models"
	sdkproject "github.com/appwrite/sdk-for-go/v7/project"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

const platformResourceName = "appwrite_project_platform"

// Platforms are project resources, authorized with a project key's
// platforms.* scopes.
var platformScopes = []string{"platforms.read", "platforms.write"}

func platformCredentialGuidance() string {
	return common.ProjectCredentialGuidance(platformResourceName, platformScopes...)
}

// Platform types Appwrite can create, each with its own create and update
// route.
const (
	platformTypeWeb     = "web"
	platformTypeAndroid = "android"
	platformTypeApple   = "apple"
	platformTypeLinux   = "linux"
	platformTypeWindows = "windows"
)

// platformKind describes what one platform type takes besides its name: a
// single identifying field, named as the API names it.
type platformKind struct {
	name      string
	attribute string
	// maxLength is in characters, as the server counts them.
	maxLength int
	what      string
}

// The server accepts identifiers of up to 256 characters but stores them in a
// 255-character column, and answers a 256-character one with a 500. 255 is the
// limit that actually holds.
var platformKinds = []platformKind{
	{name: platformTypeWeb, attribute: "hostname", maxLength: 253, what: "host name"},
	{name: platformTypeAndroid, attribute: "application_id", maxLength: 255, what: "application ID"},
	{name: platformTypeApple, attribute: "bundle_identifier", maxLength: 255, what: "bundle identifier"},
	{name: platformTypeLinux, attribute: "package_name", maxLength: 255, what: "package name"},
	{name: platformTypeWindows, attribute: "package_identifier_name", maxLength: 255, what: "package identifier name"},
}

func platformTypeNames() []string {
	names := make([]string, 0, len(platformKinds))
	for _, kind := range platformKinds {
		names = append(names, kind.name)
	}
	return names
}

func platformKindOf(platformType string) (platformKind, bool) {
	for _, kind := range platformKinds {
		if kind.name == platformType {
			return kind, true
		}
	}
	return platformKind{}, false
}

const (
	maxPlatformNameLength = 128
	maxPlatformIDLength   = 36
)

// platformIDPattern mirrors the server's rule for a custom ID: letters, digits,
// period, hyphen and underscore, not starting with one of the three symbols.
// It also keeps out "unique()", which the server would swap for a generated ID
// that then does not match the configuration.
var platformIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// hostnamePattern rejects what the server's host name check rejects: a scheme,
// a path or a port. Wildcards such as *.example.com are accepted.
var hostnamePattern = regexp.MustCompile(`^[^/:]*$`)

// textValidator mirrors the server's Text validator with requireNonBlank:
// at most maxLength characters, counted as characters rather than bytes, and
// at least one that is not whitespace, a separator or a formatting character.
// A byte count would reject names the server accepts, and validation that is
// stricter than the server cannot be relaxed later without someone noticing.
type textValidator struct {
	maxLength int
}

func (v textValidator) Description(_ context.Context) string {
	return fmt.Sprintf("must be between 1 and %d characters long and not blank", v.maxLength)
}

func (v textValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v textValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if n := utf8.RuneCountInString(value); n > v.maxLength {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Attribute Value Length",
			fmt.Sprintf("Attribute %s must be at most %d characters long, got: %d.", req.Path, v.maxLength, n))
		return
	}
	if isBlank(value) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Attribute Value",
			fmt.Sprintf("Attribute %s must not be empty or blank.", req.Path))
	}
}

func isBlank(value string) bool {
	for _, r := range value {
		if !unicode.IsSpace(r) && !unicode.In(r, unicode.Z, unicode.Cf) {
			return false
		}
	}
	return true
}

// identifierField returns the model field holding the given type's
// identifying value.
func (m *platformResourceModel) identifierField(platformType string) *types.String {
	switch platformType {
	case platformTypeWeb:
		return &m.Hostname
	case platformTypeAndroid:
		return &m.ApplicationID
	case platformTypeApple:
		return &m.BundleIdentifier
	case platformTypeLinux:
		return &m.PackageName
	case platformTypeWindows:
		return &m.PackageIdentifierName
	default:
		return nil
	}
}

// validatePlatformConfig requires the identifying argument of the configured
// type and rejects the others, which the type's route does not take.
func validatePlatformConfig(config platformResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	platformType := config.Type.ValueString()
	if _, ok := platformKindOf(platformType); !ok {
		// The type's own validator reports it.
		return diags
	}

	for _, kind := range platformKinds {
		value := *config.identifierField(kind.name)
		// A value from another resource may not be known until apply, and
		// may turn out null. Only a value known now can be judged.
		if value.IsUnknown() {
			continue
		}
		switch {
		case kind.name == platformType && value.IsNull():
			diags.AddAttributeError(path.Root(kind.attribute), "Missing required argument",
				fmt.Sprintf("%s platforms require %s, the app's %s.", platformType, kind.attribute, kind.what))
		case kind.name != platformType && !value.IsNull():
			diags.AddAttributeError(path.Root(kind.attribute), "Unsupported argument",
				fmt.Sprintf("%s is used only by %s platforms, not %s. Remove it from the configuration.",
					kind.attribute, kind.name, platformType))
		}
	}
	return diags
}

// createPlatform sends the create for the planned platform's type.
func createPlatform(client *sdkproject.Project, platformID string, plan platformResourceModel) (models.Model, error) {
	name := plan.Name.ValueString()
	value := plan.identifierField(plan.Type.ValueString())
	if value == nil {
		return nil, fmt.Errorf("unsupported platform type %q", plan.Type.ValueString())
	}
	identifier := value.ValueString()

	switch plan.Type.ValueString() {
	case platformTypeWeb:
		return client.CreateWebPlatform(platformID, name, identifier)
	case platformTypeAndroid:
		return client.CreateAndroidPlatform(platformID, name, identifier)
	case platformTypeApple:
		return client.CreateApplePlatform(platformID, name, identifier)
	case platformTypeLinux:
		return client.CreateLinuxPlatform(platformID, name, identifier)
	default:
		return client.CreateWindowsPlatform(platformID, name, identifier)
	}
}

// updatePlatform sends the update for the planned platform's type. Each
// update route replaces both the name and the identifying value.
func updatePlatform(client *sdkproject.Project, plan platformResourceModel) (models.Model, error) {
	platformID := plan.ID.ValueString()
	name := plan.Name.ValueString()
	value := plan.identifierField(plan.Type.ValueString())
	if value == nil {
		return nil, fmt.Errorf("unsupported platform type %q", plan.Type.ValueString())
	}
	identifier := value.ValueString()

	switch plan.Type.ValueString() {
	case platformTypeWeb:
		return client.UpdateWebPlatform(platformID, name, identifier)
	case platformTypeAndroid:
		return client.UpdateAndroidPlatform(platformID, name, identifier)
	case platformTypeApple:
		return client.UpdateApplePlatform(platformID, name, identifier)
	case platformTypeLinux:
		return client.UpdateLinuxPlatform(platformID, name, identifier)
	default:
		return client.UpdateWindowsPlatform(platformID, name, identifier)
	}
}

// platformFields is what every platform model has in common, flattened.
type platformFields struct {
	id, platformType, name, identifier, createdAt, updatedAt string
}

func flattenPlatform(platform models.Model) (platformFields, error) {
	switch p := platform.(type) {
	case *models.PlatformWeb:
		return platformFields{p.Id, platformTypeWeb, p.Name, p.Hostname, p.CreatedAt, p.UpdatedAt}, nil
	case *models.PlatformAndroid:
		return platformFields{p.Id, platformTypeAndroid, p.Name, p.ApplicationId, p.CreatedAt, p.UpdatedAt}, nil
	case *models.PlatformApple:
		return platformFields{p.Id, platformTypeApple, p.Name, p.BundleIdentifier, p.CreatedAt, p.UpdatedAt}, nil
	case *models.PlatformLinux:
		return platformFields{p.Id, platformTypeLinux, p.Name, p.PackageName, p.CreatedAt, p.UpdatedAt}, nil
	case *models.PlatformWindows:
		return platformFields{p.Id, platformTypeWindows, p.Name, p.PackageIdentifierName, p.CreatedAt, p.UpdatedAt}, nil
	default:
		return platformFields{}, fmt.Errorf("unexpected platform response %T", platform)
	}
}

// mapPlatformToState copies a platform into the model. The identifying
// argument of the platform's type is set and the others are null, which is
// what the configuration validation requires.
func mapPlatformToState(platform models.Model, model *platformResourceModel) error {
	fields, err := flattenPlatform(platform)
	if err != nil {
		return err
	}

	model.ID = types.StringValue(fields.id)
	model.Type = types.StringValue(fields.platformType)
	model.Name = types.StringValue(fields.name)
	model.CreatedAt = types.StringValue(fields.createdAt)
	model.UpdatedAt = types.StringValue(fields.updatedAt)
	for _, kind := range platformKinds {
		*model.identifierField(kind.name) = types.StringNull()
	}
	*model.identifierField(fields.platformType) = types.StringValue(fields.identifier)
	return nil
}

// platformErrorDetail adds what the server's message leaves out for the two
// errors a user can do something about.
func platformErrorDetail(err error) string {
	formatted := common.FormatErrorWithAuthGuidance(err, platformCredentialGuidance())
	switch common.ErrorType(err) {
	case "platform_already_exists":
		// The message says "in this project", but platform IDs are unique
		// across the whole Appwrite instance.
		return formatted + "\n\nPlatform IDs are unique across the whole Appwrite instance, not only within a " +
			"project, so another project may already use this one. Leave id unset to have one generated."
	case "platform_method_unsupported":
		// A platform created before Appwrite merged its platform types keeps
		// its original type, such as flutter-ios. It reads back as the merged
		// type, but the merged type's update route refuses it.
		return formatted + "\n\nThis platform was created with one of the platform types Appwrite has since " +
			"merged, such as flutter-ios or react-native-android. Appwrite reports it under the merged type but " +
			"cannot update it in place. Replace it instead, for example with terraform apply -replace."
	default:
		return formatted
	}
}

// platformImportID splits platform_id or project_id/platform_id.
func platformImportID(id string) (projectID, platformID string, err error) {
	const expected = "platform_id or project_id/platform_id"
	parts := strings.Split(id, "/")
	for _, part := range parts {
		if part == "" {
			return "", "", fmt.Errorf("expected %s, got: %s", expected, id)
		}
	}
	switch len(parts) {
	case 1:
		return "", parts[0], nil
	case 2:
		return parts[0], parts[1], nil
	default:
		return "", "", fmt.Errorf("expected %s, got: %s", expected, id)
	}
}
