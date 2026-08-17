package common_test

import (
	"strings"
	"testing"

	"github.com/appwrite/sdk-for-go/v6/appwrite"
	"github.com/appwrite/sdk-for-go/v6/client"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOrganizationHelpers(t *testing.T) {
	clients := &common.AppwriteClients{
		BaseOptions: []client.ClientOption{
			appwrite.WithEndpoint("https://example.com/v1"),
			appwrite.WithKey("standard_project-key"),
		},
		OrganizationBaseOptions: []client.ClientOption{
			appwrite.WithEndpoint("https://example.com/v1"),
			appwrite.WithKey("organization_org-key"),
		},
		ProjectCredentialType:      common.CredentialTypeStandard,
		OrganizationCredentialType: common.CredentialTypeOrganization,
		OrganizationID:             "provider-org",
	}

	organizationID, err := common.ResolveOrganizationID(clients, types.StringNull())
	if err != nil {
		t.Fatalf("ResolveOrganizationID returned an error: %v", err)
	}
	if organizationID != "provider-org" {
		t.Fatalf("organization ID = %q, want provider-org", organizationID)
	}

	organizationID, err = common.ResolveOrganizationID(clients, types.StringValue("resource-org"))
	if err != nil {
		t.Fatalf("ResolveOrganizationID returned an error: %v", err)
	}
	if organizationID != "resource-org" {
		t.Fatalf("organization ID = %q, want resource-org", organizationID)
	}

	organizationClient := clients.ClientForOrganization("resource-org")
	if organizationClient.Config["project"] != "console" {
		t.Errorf("project = %q, want console", organizationClient.Config["project"])
	}
	if organizationClient.Headers["X-Appwrite-Organization"] != "resource-org" {
		t.Errorf("organization header = %q, want resource-org", organizationClient.Headers["X-Appwrite-Organization"])
	}
	if organizationClient.Headers["X-Appwrite-Key"] != "organization_org-key" {
		t.Errorf("key header = %q, want organization_org-key", organizationClient.Headers["X-Appwrite-Key"])
	}

	organizationProjectClient := clients.ClientForOrganizationProject("project-id", "resource-org")
	if organizationProjectClient.Config["project"] != "project-id" {
		t.Errorf("organization project client project = %q, want project-id", organizationProjectClient.Config["project"])
	}
	if organizationProjectClient.Headers["X-Appwrite-Key"] != "organization_org-key" {
		t.Errorf("organization project client key header = %q, want organization_org-key", organizationProjectClient.Headers["X-Appwrite-Key"])
	}

	projectClient := clients.ClientForProject("project-id")
	if projectClient.Headers["X-Appwrite-Key"] != "standard_project-key" {
		t.Errorf("project client key header = %q, want standard_project-key", projectClient.Headers["X-Appwrite-Key"])
	}
	if projectClient.Headers["X-Appwrite-Organization"] != "provider-org" {
		t.Errorf("project client organization header = %q, want provider-org", projectClient.Headers["X-Appwrite-Organization"])
	}
}

func TestCredentialTypes(t *testing.T) {
	tests := map[string]common.CredentialType{
		"standard_secret":     common.CredentialTypeStandard,
		"ephemeral_token":     common.CredentialTypeEphemeral,
		"organization_secret": common.CredentialTypeOrganization,
		"account_secret":      common.CredentialTypeAccount,
		"oauth2_secret":       common.CredentialTypeOAuth2,
		"legacy-key":          common.CredentialTypeUnknown,
		"SG.legacy":           common.CredentialTypeUnknown,
		"future_secret":       common.CredentialTypeUnknown,
	}
	for key, expected := range tests {
		if actual := common.DetectCredentialType(key); actual != expected {
			t.Errorf("DetectCredentialType(%q) = %q, want %q", key, actual, expected)
		}
	}
}

func TestCredentialValidation(t *testing.T) {
	clients := &common.AppwriteClients{
		ProjectCredentialType:      common.CredentialTypeOrganization,
		OrganizationCredentialType: common.CredentialTypeStandard,
	}
	if err := common.ValidateProjectCredential(clients, "appwrite_proxy_rule", "rules.write"); err == nil || !strings.Contains(err.Error(), "APPWRITE_API_KEY") {
		t.Fatalf("expected actionable project credential error, got %v", err)
	}
	if err := common.ValidateOrganizationCredential(clients, "appwrite_project", "projects.write"); err == nil || !strings.Contains(err.Error(), "APPWRITE_ORGANIZATION_API_KEY") {
		t.Fatalf("expected actionable organization credential error, got %v", err)
	}

	clients.ProjectCredentialType = common.CredentialTypeUnknown
	clients.OrganizationCredentialType = common.CredentialTypeOrganization
	if err := common.ValidateProjectCredential(clients, "legacy"); err != nil {
		t.Fatalf("legacy project credential was rejected: %v", err)
	}
	if err := common.ValidateOrganizationCredential(clients, "appwrite_project"); err != nil {
		t.Fatalf("organization credential was rejected: %v", err)
	}
}

func TestResolveOrganizationIDMissing(t *testing.T) {
	_, err := common.ResolveOrganizationID(&common.AppwriteClients{}, types.StringNull())
	if err == nil {
		t.Fatal("ResolveOrganizationID returned nil error without an organization ID")
	}
}

func TestVariableKeyValidators(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{name: "uppercase", key: "API_URL", valid: true},
		{name: "leading underscore", key: "_PRIVATE", valid: true},
		{name: "trailing digit", key: "KEY1", valid: true},
		{name: "max length", key: strings.Repeat("A", 255), valid: true},
		{name: "hyphen", key: "MY-KEY", valid: false},
		{name: "dot", key: "MY.KEY", valid: false},
		{name: "space", key: "MY KEY", valid: false},
		{name: "leading digit", key: "9KEY", valid: false},
		{name: "accent", key: "KÉY", valid: false},
		{name: "tab", key: "KEY\t", valid: false},
		{name: "empty", key: "", valid: false},
		{name: "too long", key: strings.Repeat("A", 256), valid: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("key"),
				ConfigValue: types.StringValue(tc.key),
			}
			resp := &validator.StringResponse{}

			for _, v := range common.VariableKeyValidators() {
				v.ValidateString(t.Context(), req, resp)
			}

			if got := !resp.Diagnostics.HasError(); got != tc.valid {
				t.Errorf("key %q: got valid=%v, want valid=%v (%s)", tc.key, got, tc.valid, resp.Diagnostics)
			}
		})
	}
}
