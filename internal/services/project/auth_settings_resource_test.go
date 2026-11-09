package project_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
)

// Auth methods, policies and mock numbers share one attribute of the project
// document, and Appwrite rewrites all of it on every change, so two changes in
// flight at once lose one. Terraform applies these resources in parallel; the
// server checks are what show every change arrived.
func TestAccProjectAuthSettings_appliedTogether(t *testing.T) {
	config := func(sessionLimit, otp string) string {
		return `
resource "appwrite_project_auth_methods" "test" {
  anonymous = false
  magic_url = false
}

resource "appwrite_project_auth_policies" "test" {
  session_limit = ` + sessionLimit + `
  user_limit    = 10000

  mfa_factors = {
    custom = true
  }
}

resource "appwrite_project_mock_phone" "test" {
  count = 4

  number = "+1555555030${count.index}"
  otp    = "` + otp + `"
}
`
	}

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("30", "111111"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAuthMethodsOnServer(t, map[string]bool{
						"email-password": true, "magic-url": false, "email-otp": true, "anonymous": false,
						"invites": true, "jwt": true, "phone": true,
					}),
					checkPoliciesOnServer(t, map[string]map[string]any{
						"session-limit": {"total": 30.0},
						"user-limit":    {"total": 10000.0},
						"mfa-factors":   {"totp": true, "email": true, "phone": true, "custom": true},
					}),
					checkMockPhonesOnServer(t, 4, "111111"),
				),
			},
			// Changing all of them at once, so the updates race as well.
			{
				Config: config("40", "222222"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkPoliciesOnServer(t, map[string]map[string]any{
						"session-limit": {"total": 40.0},
					}),
					checkMockPhonesOnServer(t, 4, "222222"),
					checkAuthMethodsOnServer(t, map[string]bool{"anonymous": false, "magic-url": false}),
				),
			},
		},
	})
}
