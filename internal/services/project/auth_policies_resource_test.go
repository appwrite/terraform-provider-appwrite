package project_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
)

const authPolicies = "appwrite_project_auth_policies.test"

// TestAccProjectAuthPolicies_sessionsAndTeams changes the policies that cannot
// affect the other tests sharing the project: none of them is checked when a
// user is created with an API key. The password and sign-up email policies are
// in TestAccProjectAuthPolicies_userValidation, which needs a project of its
// own.
func TestAccProjectAuthPolicies_sessionsAndTeams(t *testing.T) {
	changed := `
resource "appwrite_project_auth_policies" "test" {
  password_history     = 5
  session_alert        = true
  session_duration     = 3600
  session_invalidation = false
  session_limit        = 50
  user_limit           = 10000

  membership_privacy = {
    user_id          = true
    user_name        = true
    user_accessed_at = true
  }

  mfa_factors = {
    phone  = false
    custom = true
  }
}
`
	// Everything left out returns to its default, including password_history,
	// which the API only turns off with null.
	reduced := `
resource "appwrite_project_auth_policies" "test" {
  session_limit = 20

  mfa_factors = {
    custom = true
  }
}
`

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: changed,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(authPolicies, "id", os.Getenv("APPWRITE_PROJECT_ID")),
					resource.TestCheckResourceAttr(authPolicies, "password_history", "5"),
					resource.TestCheckResourceAttr(authPolicies, "session_duration", "3600"),
					resource.TestCheckResourceAttr(authPolicies, "session_invalidation", "false"),
					resource.TestCheckResourceAttr(authPolicies, "membership_privacy.user_id", "true"),
					resource.TestCheckResourceAttr(authPolicies, "membership_privacy.user_email", "false"),
					resource.TestCheckResourceAttr(authPolicies, "mfa_factors.totp", "true"),
					resource.TestCheckResourceAttr(authPolicies, "mfa_factors.phone", "false"),
					// Left out, so at their defaults.
					resource.TestCheckResourceAttr(authPolicies, "password_strength.min", "8"),
					resource.TestCheckResourceAttr(authPolicies, "password_pwned.enabled", "true"),
					resource.TestCheckResourceAttr(authPolicies, "deny_corporate_email", "false"),
					checkPoliciesOnServer(t, map[string]map[string]any{
						"password-history":     {"total": 5.0},
						"session-alert":        {"enabled": true},
						"session-duration":     {"duration": 3600.0},
						"session-invalidation": {"enabled": false},
						"session-limit":        {"total": 50.0},
						"user-limit":           {"total": 10000.0},
						"membership-privacy": {
							"userId": true, "userEmail": false, "userPhone": false,
							"userName": true, "userMFA": false, "userAccessedAt": true,
						},
						"mfa-factors": {"totp": true, "email": true, "phone": false, "custom": true},
					}),
				),
			},
			{
				ResourceName:      authPolicies,
				ImportState:       true,
				ImportStateId:     os.Getenv("APPWRITE_PROJECT_ID"),
				ImportStateVerify: true,
			},
			{
				Config: reduced,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(authPolicies, "password_history", "0"),
					resource.TestCheckResourceAttr(authPolicies, "session_limit", "20"),
					resource.TestCheckResourceAttr(authPolicies, "mfa_factors.phone", "true"),
					checkPoliciesOnServer(t, map[string]map[string]any{
						"password-history":     {"total": 0.0},
						"session-alert":        {"enabled": false},
						"session-duration":     {"duration": 31536000.0},
						"session-invalidation": {"enabled": true},
						"session-limit":        {"total": 20.0},
						"user-limit":           {"total": 0.0},
						"mfa-factors":          {"totp": true, "email": true, "phone": true, "custom": true},
					}),
				),
			},
			acceptance.ExpectEmptyPlanAfterApply(reduced),
		},
	})
}

// TestAccProjectAuthPolicies_userValidation changes the policies Appwrite
// checks whenever a user is created, which would reject the users the other
// tests create in a shared project.
func TestAccProjectAuthPolicies_userValidation(t *testing.T) {
	config := `
resource "appwrite_project_auth_policies" "test" {
  password_dictionary    = true
  password_personal_data = true

  password_strength = {
    min       = 12
    uppercase = true
    symbols   = true
  }

  deny_aliased_email    = true
  deny_disposable_email = true
  deny_free_email       = true
  deny_corporate_email  = true
}
`
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.ExclusiveProjectPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(authPolicies, "password_strength.min", "12"),
					resource.TestCheckResourceAttr(authPolicies, "password_strength.lowercase", "false"),
					checkPoliciesOnServer(t, map[string]map[string]any{
						"password-dictionary":    {"enabled": true},
						"password-personal-data": {"enabled": true},
						"password-strength":      {"min": 12.0, "uppercase": true, "lowercase": false, "number": false, "symbols": true},
						"deny-aliased-email":     {"enabled": true},
						"deny-disposable-email":  {"enabled": true},
						"deny-free-email":        {"enabled": true},
						"deny-corporate-email":   {"enabled": true},
					}),
				),
			},
			{
				ResourceName:      authPolicies,
				ImportState:       true,
				ImportStateId:     os.Getenv("APPWRITE_PROJECT_ID"),
				ImportStateVerify: true,
			},
		},
	})
}

// checkPoliciesOnServer compares the named policies as Appwrite reports them,
// independently of the resource's own Read.
func checkPoliciesOnServer(t *testing.T, want map[string]map[string]any) resource.TestCheckFunc {
	return func(*terraform.State) error {
		list, err := appwrite.NewProject(acceptance.ProjectClient(t)).ListPolicies()
		if err != nil {
			return err
		}
		var decoded struct {
			Policies []map[string]any `json:"policies"`
		}
		if err := list.Decode(&decoded); err != nil {
			return err
		}
		got := map[string]map[string]any{}
		for _, policy := range decoded.Policies {
			id, _ := policy["$id"].(string)
			delete(policy, "$id")
			got[id] = policy
		}
		for id, fields := range want {
			if !reflect.DeepEqual(got[id], fields) {
				return fmt.Errorf("policy %s is %v on the server, want %v", id, got[id], fields)
			}
		}
		return nil
	}
}
