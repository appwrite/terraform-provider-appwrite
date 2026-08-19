package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// CredentialType identifies the kind of Appwrite API key configured on the
// provider. Legacy keys without a type prefix are left unknown and validated
// by the server for backwards compatibility.
type CredentialType string

const (
	CredentialTypeUnknown      CredentialType = "unknown"
	CredentialTypeStandard     CredentialType = "standard"
	CredentialTypeEphemeral    CredentialType = "ephemeral"
	CredentialTypeOrganization CredentialType = "organization"
	CredentialTypeAccount      CredentialType = "account"
	CredentialTypeOAuth2       CredentialType = "oauth2"
)

// AppwriteClients holds the base configuration for creating SDK clients.
type AppwriteClients struct {
	// BaseOptions contains the endpoint and project API key options.
	BaseOptions []client.ClientOption
	// OrganizationBaseOptions contains the endpoint and organization API key
	// options. It falls back to BaseOptions for backwards compatibility.
	OrganizationBaseOptions []client.ClientOption
	// ProjectCredentialType is the detected type of api_key.
	ProjectCredentialType CredentialType
	// OrganizationCredentialType is the detected type of organization_api_key,
	// or api_key when no dedicated organization key is configured.
	OrganizationCredentialType CredentialType
	// ProjectID is the provider-level default project ID.
	ProjectID string
	// OrganizationID is the provider-level default organization ID.
	OrganizationID string
}

// WithUserAgent returns a ClientOption that sets the User-Agent header to identify
// Terraform provider traffic. This is required for HashiCorp partner providers.
func WithUserAgent(version string) client.ClientOption {
	return func(clt *client.Client) error {
		clt.Headers["user-agent"] = fmt.Sprintf("terraform-provider-appwrite/%s", version)
		return nil
	}
}

// ClientForProject creates a new SDK client targeting a specific project.
func (ac *AppwriteClients) ClientForProject(projectID string) client.Client {
	opts := make([]client.ClientOption, 0, len(ac.BaseOptions)+1)
	opts = append(opts, ac.BaseOptions...)
	opts = append(opts, appwrite.WithProject(projectID))
	c := appwrite.NewClient(opts...)
	if ac.OrganizationID != "" {
		c.AddHeader("X-Appwrite-Organization", ac.OrganizationID)
	}
	return c
}

// ClientForOrganization creates a client targeting the console project and
// scopes it to an organization. Organization routes have no organization ID in
// their path, so Appwrite resolves it from X-Appwrite-Organization.
func (ac *AppwriteClients) ClientForOrganization(organizationID string) client.Client {
	return ac.clientWithOrganizationCredential("console", organizationID)
}

// ClientForOrganizationProject creates a project-targeted client authenticated
// with the organization credential. Console administration routes such as
// project API key management need both the project and organization headers.
func (ac *AppwriteClients) ClientForOrganizationProject(projectID, organizationID string) client.Client {
	return ac.clientWithOrganizationCredential(projectID, organizationID)
}

func (ac *AppwriteClients) clientWithOrganizationCredential(projectID, organizationID string) client.Client {
	baseOptions := ac.OrganizationBaseOptions
	if len(baseOptions) == 0 {
		baseOptions = ac.BaseOptions
	}
	opts := make([]client.ClientOption, 0, len(baseOptions)+1)
	opts = append(opts, baseOptions...)
	opts = append(opts, appwrite.WithProject(projectID))
	c := appwrite.NewClient(opts...)
	c.AddHeader("X-Appwrite-Organization", organizationID)
	return c
}

// DetectCredentialType returns the type prefix of a modern Appwrite API key.
// Unknown and legacy key formats are intentionally accepted so the server can
// validate them.
func DetectCredentialType(apiKey string) CredentialType {
	prefix, _, ok := strings.Cut(apiKey, "_")
	if !ok {
		return CredentialTypeUnknown
	}
	switch CredentialType(prefix) {
	case CredentialTypeStandard, CredentialTypeEphemeral, CredentialTypeOrganization, CredentialTypeAccount, CredentialTypeOAuth2:
		return CredentialType(prefix)
	default:
		return CredentialTypeUnknown
	}
}

// ValidateProjectCredential rejects credentials that are known not to work on
// project-scoped server routes. Unknown keys are allowed for compatibility with
// legacy Appwrite key formats.
func ValidateProjectCredential(clients *AppwriteClients, resourceName string, scopes ...string) error {
	switch clients.ProjectCredentialType {
	case CredentialTypeOrganization, CredentialTypeAccount, CredentialTypeOAuth2:
		return fmt.Errorf("%s; configured api_key has credential type %q", strings.TrimSuffix(ProjectCredentialGuidance(resourceName, scopes...), "."), clients.ProjectCredentialType)
	default:
		return nil
	}
}

// ValidateOrganizationCredential rejects credentials that are known not to
// work on organization administration routes. Unknown keys are allowed for
// compatibility with legacy Appwrite key formats.
func ValidateOrganizationCredential(clients *AppwriteClients, resourceName string, scopes ...string) error {
	switch clients.OrganizationCredentialType {
	case CredentialTypeStandard, CredentialTypeEphemeral, CredentialTypeAccount, CredentialTypeOAuth2:
		return fmt.Errorf("%s; configured organization credential has type %q", strings.TrimSuffix(OrganizationCredentialGuidance(resourceName, scopes...), "."), clients.OrganizationCredentialType)
	default:
		return nil
	}
}

// ProjectCredentialGuidance describes the credential required by a
// project-scoped resource without exposing any credential value.
func ProjectCredentialGuidance(resourceName string, scopes ...string) string {
	return fmt.Sprintf("%s requires a standard or ephemeral project API key%s. Configure it with api_key or APPWRITE_API_KEY.", resourceName, formatRequiredScopes(scopes))
}

// OrganizationCredentialGuidance describes the credential required by an
// organization administration resource without exposing any credential value.
func OrganizationCredentialGuidance(resourceName string, scopes ...string) string {
	return fmt.Sprintf("%s requires an organization API key%s. Configure it with organization_api_key or APPWRITE_ORGANIZATION_API_KEY.", resourceName, formatRequiredScopes(scopes))
}

func formatRequiredScopes(scopes []string) string {
	if len(scopes) == 0 {
		return ""
	}
	return fmt.Sprintf(" with the %s scopes", strings.Join(scopes, " and "))
}

// ResolveProjectID returns the resource-level project_id if set, otherwise the provider default.
// Returns an error if neither is set.
func ResolveProjectID(clients *AppwriteClients, resourceProjectID types.String) (string, error) {
	if !resourceProjectID.IsNull() && !resourceProjectID.IsUnknown() && resourceProjectID.ValueString() != "" {
		return resourceProjectID.ValueString(), nil
	}
	if clients.ProjectID != "" {
		return clients.ProjectID, nil
	}
	return "", fmt.Errorf("project_id must be set either on the provider or the resource")
}

// ResolveOrganizationID returns the resource-level organization_id if set,
// otherwise the provider default. Returns an error if neither is set.
func ResolveOrganizationID(clients *AppwriteClients, resourceOrganizationID types.String) (string, error) {
	if !resourceOrganizationID.IsNull() && !resourceOrganizationID.IsUnknown() && resourceOrganizationID.ValueString() != "" {
		return resourceOrganizationID.ValueString(), nil
	}
	if clients.OrganizationID != "" {
		return clients.OrganizationID, nil
	}
	return "", fmt.Errorf("organization_id must be set either on the provider or the resource")
}

// variableKeyPattern mirrors the API rule for variable keys: they become
// environment variable names at build and runtime, so only C-style identifiers
// are accepted.
var variableKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// VariableKeyValidators returns the validators for a variable key, so an
// unusable key is reported at plan time instead of failing the apply.
func VariableKeyValidators() []validator.String {
	return []validator.String{
		stringvalidator.LengthAtMost(255),
		stringvalidator.RegexMatches(
			variableKeyPattern,
			"must contain only letters, digits and underscores, and must not start with a digit",
		),
	}
}

// ProjectIDAttribute returns the shared schema attribute for project_id on resources.
func ProjectIDAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Description:   "The Appwrite project ID. Defaults to the provider-level project_id.",
		Optional:      true,
		Computed:      true,
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
	}
}

// OrganizationIDAttribute returns the shared schema attribute for
// organization_id on organization-scoped resources.
func OrganizationIDAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Description:   "The Appwrite organization ID. Defaults to the provider-level organization_id.",
		Optional:      true,
		Computed:      true,
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
	}
}

// IsNotFoundError checks if an Appwrite SDK error is a 404.
func IsNotFoundError(err error) bool {
	var appErr *client.AppwriteError
	if errors.As(err, &appErr) {
		return appErr.GetStatusCode() == 404
	}
	return false
}

// IsColumnNotAvailableError checks if the error is due to a column still being processed.
func IsColumnNotAvailableError(err error) bool {
	var appErr *client.AppwriteError
	if errors.As(err, &appErr) {
		return appErr.GetStatusCode() == 400 && strings.Contains(appErr.GetMessage(), "not yet available")
	}
	return false
}

// FormatError returns a detailed error string including status code and response body.
func FormatError(err error) string {
	var appErr *client.AppwriteError
	if errors.As(err, &appErr) {
		return fmt.Sprintf("%s (status: %d, response: %s)", appErr.GetMessage(), appErr.GetStatusCode(), appErr.GetResponse())
	}
	return err.Error()
}

// FormatErrorWithAuthGuidance appends resource-specific credential guidance to
// authentication and authorization failures returned by Appwrite.
func FormatErrorWithAuthGuidance(err error, guidance string) string {
	formatted := FormatError(err)
	var appErr *client.AppwriteError
	if !errors.As(err, &appErr) || guidance == "" {
		return formatted
	}

	var response struct {
		Type string `json:"type"`
	}
	if json.Unmarshal([]byte(appErr.GetResponse()), &response) != nil {
		return formatted
	}
	switch response.Type {
	case "general_unauthorized_scope", "user_unauthorized", "key_creation_denied":
		return formatted + "\n\nAuthentication guidance: " + guidance
	default:
		return formatted
	}
}

// GetColumnRaw fetches a column using a raw API call, bypassing the SDK's
// type-matching logic which doesn't handle all column types (e.g. text,
// longtext, mediumtext, varchar, point, line, polygon).
func GetColumnRaw(c client.Client, databaseID, tableID, key string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/tablesdb/%s/tables/%s/columns/%s", databaseID, tableID, key)
	resp, err := c.Call("GET", path, map[string]interface{}{
		// The Appwrite Go SDK stores the project id in client.Config["project"]
		// (set by appwrite.WithProject) and only injects the X-Appwrite-Project
		// header inside each typed service method. client.Call does not add it
		// from Config, so this raw call must pass it explicitly or the server
		// rejects the request with project_id_missing (HTTP 403). The typed
		// table/index reads work because they set this header.
		"X-Appwrite-Project": c.Config["project"],
	}, map[string]interface{}{
		"databaseId": databaseID,
		"tableId":    tableID,
		"key":        key,
	})
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(resp.Type, "application/json") {
		return nil, fmt.Errorf("unexpected response type: %s", resp.Type)
	}
	// The SDK hands back []byte for JSON responses since v6.5.0, and string
	// before that; ResponseBody accepts either.
	body, err := client.ResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read column response: %w", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal column response: %w", err)
	}
	return result, nil
}

// AttrCheck holds the result of an attribute mismatch check.
type AttrCheck struct {
	Summary  string
	Detail   string
	Mismatch bool
}

// CheckBoolNotIgnored returns an error diagnostic if the planned bool value differs from
// the API response, indicating the server doesn't support this attribute.
func CheckBoolNotIgnored(planned types.Bool, actual bool, attrName string, resourceDesc string) AttrCheck {
	if planned.IsNull() || planned.IsUnknown() {
		return AttrCheck{}
	}
	if planned.ValueBool() != actual {
		return AttrCheck{
			Summary: fmt.Sprintf("Attribute %q not supported", attrName),
			Detail: fmt.Sprintf(
				"The server did not accept the %q setting for %s. "+
					"This feature may not be supported on this Appwrite server version. "+
					"Remove the %q attribute from your configuration or upgrade your server.",
				attrName, resourceDesc, attrName,
			),
			Mismatch: true,
		}
	}
	return AttrCheck{}
}

// CheckStringNotIgnored returns an error diagnostic if the planned string value differs from
// the API response, indicating the server doesn't support this attribute.
func CheckStringNotIgnored(planned types.String, actual string, attrName string, resourceDesc string) AttrCheck {
	if planned.IsNull() || planned.IsUnknown() {
		return AttrCheck{}
	}
	if planned.ValueString() != actual {
		return AttrCheck{
			Summary: fmt.Sprintf("Attribute %q not supported", attrName),
			Detail: fmt.Sprintf(
				"The server did not accept the %q setting for %s (sent %q, got %q). "+
					"This feature may not be supported on this Appwrite server version. "+
					"Remove the %q attribute from your configuration or upgrade your server.",
				attrName, resourceDesc, planned.ValueString(), actual, attrName,
			),
			Mismatch: true,
		}
	}
	return AttrCheck{}
}

// ImportColumnState parses a "database_id/table_id/key" import ID into state.
func ImportColumnState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: database_id/table_id/key, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("table_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), parts[2])...)
}

// WaitForDeploymentReady polls a deployment until its status becomes "ready",
// "failed", or "canceled". Returns nil on "ready", error otherwise.
func WaitForDeploymentReady(ctx context.Context, getDeployment func() (string, error), deploymentID string) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for deployment %q to become ready", deploymentID)
		default:
		}

		status, err := getDeployment()
		if err != nil {
			return fmt.Errorf("error checking deployment %q status: %w", deploymentID, err)
		}

		switch status {
		case "ready":
			return nil
		case "failed":
			return fmt.Errorf("deployment %q failed", deploymentID)
		case "canceled":
			return fmt.Errorf("deployment %q was canceled", deploymentID)
		}

		time.Sleep(2 * time.Second)
	}
}

// WaitForColumnAvailable polls a column until its status becomes "available",
// with a maximum wait of 5 minutes.
func WaitForColumnAvailable(ctx context.Context, getColumn func() (interface{}, error), key string) error {
	deadline := time.After(5 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for column %q to become available", key)
		case <-deadline:
			return fmt.Errorf("column %q did not become available within 5m", key)
		default:
		}

		raw, err := getColumn()
		if err != nil {
			return fmt.Errorf("error checking column %q status: %w", key, err)
		}

		var status string
		if m, ok := raw.(map[string]interface{}); ok {
			status, _ = m["status"].(string)
		}
		switch status {
		case "available":
			return nil
		case "failed", "stuck":
			return fmt.Errorf("column %q is in %q state", key, status)
		}

		time.Sleep(1 * time.Second)
	}
}

// DatabaseProductGuidance describes the API key scopes a DocumentsDB or
// VectorsDB resource needs.
//
// These products are gated behind the legacy Databases scopes (collections.*,
// documents.*) rather than the TablesDB ones (tables.*, rows.*), which is not
// obvious from their names and which the Appwrite Console does not always
// surface. Where the Console offers no such checkbox the scopes can still be
// set through the project keys API, since a key's scope list accepts names the
// Console does not render.
func DatabaseProductGuidance(resourceName string, scopes ...string) string {
	return fmt.Sprintf(
		"%s requires a project API key%s. These are the legacy Databases scopes, not the TablesDB ones; "+
			"if the Appwrite Console does not offer them, set them on the key through the project keys API instead.",
		resourceName, formatRequiredScopes(scopes),
	)
}
