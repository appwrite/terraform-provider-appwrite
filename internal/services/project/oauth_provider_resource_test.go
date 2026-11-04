package project_test

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/client"
	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Each test configures providers no other test touches, because they all share
// the test project and a provider is a project-wide setting.
//
// Enabling GitHub, Microsoft, Keycloak and a few others makes Appwrite check
// the credentials with the provider itself, so those are only configured
// disabled here. Amazon, OIDC, Okta and Apple are enabled without a check.

func TestAccProjectOAuthProvider_basic(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_project_oauth_provider" "amazon" {
  provider_id = "amazon"
  endpoint    = "example.com"
}
`,
				ExpectError: regexp.MustCompile(`endpoint is not an argument of the amazon provider`),
			},
			{
				Config: `
resource "appwrite_project_oauth_provider" "amazon" {
  provider_id   = "amazon"
  client_id     = "amzn1.application-oa2-client.tf-acc-test"
  client_secret = "tf-acc-test-secret"
}

resource "appwrite_project_oauth_provider" "github" {
  provider_id   = "github"
  enabled       = false
  client_id     = "Iv1.tf-acc-test"
  client_secret = "tf-acc-test-secret"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.amazon", "id", "amazon"),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.amazon", "enabled", "true"),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.amazon", "client_id", "amzn1.application-oa2-client.tf-acc-test"),
					resource.TestCheckResourceAttrSet("appwrite_project_oauth_provider.amazon", "project_id"),
					resource.TestCheckNoResourceAttr("appwrite_project_oauth_provider.amazon", "prompt.#"),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.github", "enabled", "false"),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.github", "client_id", "Iv1.tf-acc-test"),
				),
			},
			{
				ResourceName:            "appwrite_project_oauth_provider.amazon",
				ImportState:             true,
				ImportStateId:           "amazon",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"client_secret"},
			},
			{
				// Switch to the write-only secret and disable the provider.
				Config: `
resource "appwrite_project_oauth_provider" "amazon" {
  provider_id              = "amazon"
  enabled                  = false
  client_id                = "amzn1.application-oa2-client.tf-acc-test-2"
  client_secret_wo         = "tf-acc-test-secret-2"
  client_secret_wo_version = 1
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.amazon", "enabled", "false"),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.amazon", "client_id", "amzn1.application-oa2-client.tf-acc-test-2"),
					resource.TestCheckNoResourceAttr("appwrite_project_oauth_provider.amazon", "client_secret"),
					resource.TestCheckNoResourceAttr("appwrite_project_oauth_provider.amazon", "client_secret_wo"),
				),
			},
			{
				// Removing client_id clears it on the server.
				Config: `
resource "appwrite_project_oauth_provider" "amazon" {
  provider_id = "amazon"
  enabled     = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("appwrite_project_oauth_provider.amazon", "client_id"),
					checkOAuthProviderField(t, "amazon", "clientId", ""),
				),
			},
		},
	})
}

func TestAccProjectOAuthProvider_oidc(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_project_oauth_provider" "oidc" {
  provider_id    = "oidc"
  client_id      = "tf-acc-test"
  client_secret  = "tf-acc-test-secret"
  well_known_url = "https://idp.example.com/.well-known/openid-configuration"
  prompt         = ["login", "consent"]
  max_age        = 3600
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.oidc", "enabled", "true"),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.oidc", "well_known_url", "https://idp.example.com/.well-known/openid-configuration"),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.oidc", "prompt.#", "2"),
					resource.TestCheckTypeSetElemAttr("appwrite_project_oauth_provider.oidc", "prompt.*", "login"),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.oidc", "max_age", "3600"),
					resource.TestCheckNoResourceAttr("appwrite_project_oauth_provider.oidc", "authorization_url"),
				),
			},
			{
				ResourceName:            "appwrite_project_oauth_provider.oidc",
				ImportState:             true,
				ImportStateIdFunc:       importIDWithProject("appwrite_project_oauth_provider.oidc"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"client_secret"},
			},
			{
				// Replace discovery with explicit endpoints. Removing
				// well_known_url clears it, and the omitted prompt and max_age
				// keep the values Appwrite holds.
				Config: `
resource "appwrite_project_oauth_provider" "oidc" {
  provider_id       = "oidc"
  client_id         = "tf-acc-test"
  client_secret     = "tf-acc-test-secret"
  authorization_url = "https://idp.example.com/oauth2/authorize"
  token_url         = "https://idp.example.com/oauth2/token"
  user_info_url     = "https://idp.example.com/oauth2/userinfo"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("appwrite_project_oauth_provider.oidc", "well_known_url"),
					checkOAuthProviderField(t, "oidc", "wellKnownURL", ""),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.oidc", "token_url", "https://idp.example.com/oauth2/token"),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.oidc", "prompt.#", "2"),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.oidc", "max_age", "3600"),
				),
			},
		},
	})
}

func TestAccProjectOAuthProvider_providerSettings(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_project_oauth_provider" "okta" {
  provider_id             = "okta"
  client_id               = "tf-acc-test"
  client_secret           = "tf-acc-test-secret"
  domain                  = "trial-6400025.okta.com"
  authorization_server_id = "aus000000000000000h7z"
}

resource "appwrite_project_oauth_provider" "keycloak" {
  provider_id   = "keycloak"
  enabled       = false
  client_id     = "tf-acc-test"
  client_secret = "tf-acc-test-secret"
  endpoint      = "keycloak.example.com"
  realm_name    = "tf-acc-test"
}

resource "appwrite_project_oauth_provider" "apple" {
  provider_id       = "apple"
  client_id         = "com.example.tf-acc-test.web"
  client_secret     = "-----BEGIN PRIVATE KEY-----\ntf-acc-test\n-----END PRIVATE KEY-----"
  key_id            = "P4000000N8"
  team_id           = "D4000000R6"
  native_client_ids = ["com.example.tf-acc-test"]
  native_enabled    = true
}

resource "appwrite_project_oauth_provider" "google" {
  provider_id = "google"
  enabled     = false
  client_id   = "tf-acc-test.apps.googleusercontent.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.okta", "enabled", "true"),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.okta", "domain", "trial-6400025.okta.com"),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.okta", "authorization_server_id", "aus000000000000000h7z"),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.keycloak", "realm_name", "tf-acc-test"),
					checkOAuthProviderField(t, "keycloak", "endpoint", "keycloak.example.com"),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.apple", "enabled", "true"),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.apple", "native_enabled", "true"),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.apple", "key_id", "P4000000N8"),
					checkOAuthProviderField(t, "apple", "serviceId", "com.example.tf-acc-test.web"),
					// Unset, prompt is read back from Appwrite: its default,
					// consent, on a fresh project, or whatever an earlier run
					// left, since destroy cannot clear it.
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.google", "prompt.#", "1"),
				),
			},
			{
				ResourceName:            "appwrite_project_oauth_provider.apple",
				ImportState:             true,
				ImportStateId:           "apple",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"client_secret"},
			},
			{
				Config: `
resource "appwrite_project_oauth_provider" "okta" {
  provider_id   = "okta"
  client_id     = "tf-acc-test"
  client_secret = "tf-acc-test-secret"
  domain        = "trial-6400025.okta.com"
}

resource "appwrite_project_oauth_provider" "keycloak" {
  provider_id   = "keycloak"
  enabled       = false
  client_id     = "tf-acc-test"
  client_secret = "tf-acc-test-secret"
  endpoint      = "keycloak.example.com"
  realm_name    = "tf-acc-test-2"
}

resource "appwrite_project_oauth_provider" "apple" {
  provider_id   = "apple"
  client_id     = "com.example.tf-acc-test.web"
  client_secret = "-----BEGIN PRIVATE KEY-----\ntf-acc-test\n-----END PRIVATE KEY-----"
  key_id        = "P4000000N8"
  team_id       = "D4000000R6"
}

resource "appwrite_project_oauth_provider" "google" {
  provider_id = "google"
  enabled     = false
  client_id   = "tf-acc-test.apps.googleusercontent.com"
  prompt      = ["select_account"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("appwrite_project_oauth_provider.okta", "authorization_server_id"),
					checkOAuthProviderField(t, "okta", "authorizationServerId", ""),
					resource.TestCheckResourceAttr("appwrite_project_oauth_provider.keycloak", "realm_name", "tf-acc-test-2"),
					resource.TestCheckNoResourceAttr("appwrite_project_oauth_provider.apple", "native_enabled"),
					resource.TestCheckNoResourceAttr("appwrite_project_oauth_provider.apple", "native_client_ids.#"),
					checkOAuthProviderField(t, "apple", "nativeEnabled", false),
					resource.TestCheckTypeSetElemAttr("appwrite_project_oauth_provider.google", "prompt.*", "select_account"),
				),
			},
		},
	})
}

// checkOAuthProviderField asserts a value on the server rather than in state,
// for the cases where state alone cannot show that a cleared argument was
// actually cleared.
func checkOAuthProviderField(t *testing.T, providerID, key string, want any) resource.TestCheckFunc {
	t.Helper()
	return func(*terraform.State) error {
		c := acceptance.ProjectClient(t)
		resp, err := c.Call("GET", "/project/oauth2/"+providerID, map[string]interface{}{
			"X-Appwrite-Project": c.Config["project"],
		}, map[string]interface{}{})
		if err != nil {
			return fmt.Errorf("reading %s: %w", providerID, err)
		}
		raw, err := client.ResponseBody(resp)
		if err != nil {
			return err
		}
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			return err
		}
		if got := body[key]; got != want {
			return fmt.Errorf("%s %s on the server is %#v, want %#v", providerID, key, got, want)
		}
		return nil
	}
}

func importIDWithProject(name string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return "", fmt.Errorf("%s not in state", name)
		}
		projectID := rs.Primary.Attributes["project_id"]
		if projectID == "" {
			projectID = os.Getenv("APPWRITE_PROJECT_ID")
		}
		return projectID + "/" + rs.Primary.Attributes["provider_id"], nil
	}
}
