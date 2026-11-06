package project_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
)

// Requests made with an API key are exempt from the auth method checks, so
// turning methods off here does not affect the other tests sharing the
// project.
func TestAccProjectAuthMethods_basic(t *testing.T) {
	allOff := `
resource "appwrite_project_auth_methods" "test" {
  email_password = false
  magic_url      = false
  email_otp      = false
  anonymous      = false
  invites        = false
  jwt            = false
  phone          = false
}
`
	// Leaving a method out sets it back to its default, enabled.
	oneOff := `
resource "appwrite_project_auth_methods" "test" {
  anonymous = false
}
`

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: allOff,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_project_auth_methods.test", "id", os.Getenv("APPWRITE_PROJECT_ID")),
					resource.TestCheckResourceAttr("appwrite_project_auth_methods.test", "project_id", os.Getenv("APPWRITE_PROJECT_ID")),
					resource.TestCheckResourceAttr("appwrite_project_auth_methods.test", "email_password", "false"),
					resource.TestCheckResourceAttr("appwrite_project_auth_methods.test", "phone", "false"),
					checkAuthMethodsOnServer(t, map[string]bool{
						"email-password": false, "magic-url": false, "email-otp": false, "anonymous": false,
						"invites": false, "jwt": false, "phone": false,
					}),
				),
			},
			{
				ResourceName:      "appwrite_project_auth_methods.test",
				ImportState:       true,
				ImportStateId:     os.Getenv("APPWRITE_PROJECT_ID"),
				ImportStateVerify: true,
			},
			{
				Config: oneOff,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_project_auth_methods.test", "anonymous", "false"),
					resource.TestCheckResourceAttr("appwrite_project_auth_methods.test", "email_password", "true"),
					resource.TestCheckResourceAttr("appwrite_project_auth_methods.test", "jwt", "true"),
					checkAuthMethodsOnServer(t, map[string]bool{
						"email-password": true, "magic-url": true, "email-otp": true, "anonymous": false,
						"invites": true, "jwt": true, "phone": true,
					}),
				),
			},
			acceptance.ExpectEmptyPlanAfterApply(oneOff),
		},
	})
}

// checkAuthMethodsOnServer asks Appwrite directly, so the test does not rely
// on the resource's own Read to say the write happened.
func checkAuthMethodsOnServer(t *testing.T, want map[string]bool) resource.TestCheckFunc {
	return func(*terraform.State) error {
		project, err := appwrite.NewProject(acceptance.ProjectClient(t)).Get()
		if err != nil {
			return err
		}
		got := map[string]bool{}
		for _, method := range project.AuthMethods {
			got[method.Id] = method.Enabled
		}
		for id, enabled := range want {
			if got[id] != enabled {
				return fmt.Errorf("auth method %s is enabled=%t on the server, want %t", id, got[id], enabled)
			}
		}
		return nil
	}
}
