package project_test

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Appwrite removed the create-project-key endpoint, so a plan that would create
// one has to fail at apply time with guidance rather than silently creating the
// wrong kind of key.
func TestAccProjectKeyResource_createUnsupported(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acceptance.OrganizationPreCheck(t)
			acceptance.PreCheck(t)
		},
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProjectKeyConfig("tf-unsupported", "Terraform Key Test", []string{"users.read"}),
				ExpectError: regexp.MustCompile(`Creating project API keys is not supported`),
			},
		},
	})
}

// Existing keys are still fully manageable once imported. The key has to be
// created out of band, so this test only runs when one is provided.
func TestAccProjectKeyResource_importAndUpdate(t *testing.T) {
	keyID := os.Getenv("APPWRITE_PROJECT_KEY_ID")
	projectID := os.Getenv("APPWRITE_PROJECT_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acceptance.OrganizationPreCheck(t)
			acceptance.PreCheck(t)
			if keyID == "" {
				t.Skip("APPWRITE_PROJECT_KEY_ID must be set to an existing project API key for this test")
			}
		},
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             testAccProjectKeyConfig(keyID, "Terraform Key Test", []string{"users.read"}),
				ResourceName:       "appwrite_project_key.test",
				ImportState:        true,
				ImportStateId:      projectID + "/" + keyID,
				ImportStatePersist: true,
				// The secret is only returned at creation time, so an imported
				// key never has one to compare.
				ImportStateVerifyIgnore: []string{"secret"},
			},
			{
				Config: testAccProjectKeyConfig(keyID, "Terraform Key Test Updated", []string{"users.read", "users.write"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_project_key.test", "id", keyID),
					resource.TestCheckResourceAttr("appwrite_project_key.test", "name", "Terraform Key Test Updated"),
					resource.TestCheckResourceAttr("appwrite_project_key.test", "scopes.#", "2"),
					resource.TestCheckResourceAttrSet("appwrite_project_key.test", "created_at"),
				),
			},
		},
	})
}

func testAccProjectKeyConfig(keyID, name string, scopes []string) string {
	quotedScopes := make([]string, len(scopes))
	for i, scope := range scopes {
		quotedScopes[i] = strconv.Quote(scope)
	}
	return fmt.Sprintf(`
resource "appwrite_project_key" "test" {
  id     = %q
  name   = %q
  scopes = [%s]
}
`, keyID, name, strings.Join(quotedScopes, ", "))
}
