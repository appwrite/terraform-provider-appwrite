package project_test

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProjectKeyResource_basic(t *testing.T) {
	keyID := fmt.Sprintf("tf-%d", time.Now().UnixNano())
	projectID := os.Getenv("APPWRITE_PROJECT_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acceptance.OrganizationPreCheck(t)
			acceptance.PreCheck(t)
		},
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectKeyConfig(keyID, "Terraform Key Test", []string{"users.read"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_project_key.test", "id", keyID),
					resource.TestCheckResourceAttr("appwrite_project_key.test", "name", "Terraform Key Test"),
					resource.TestCheckResourceAttr("appwrite_project_key.test", "scopes.#", "1"),
					resource.TestCheckResourceAttrSet("appwrite_project_key.test", "secret"),
					resource.TestCheckResourceAttrSet("appwrite_project_key.test", "created_at"),
				),
			},
			{
				ResourceName:            "appwrite_project_key.test",
				ImportState:             true,
				ImportStateId:           projectID + "/" + keyID,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret"},
			},
			{
				Config: testAccProjectKeyConfig(keyID, "Terraform Key Test Updated", []string{"users.read", "users.write"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_project_key.test", "name", "Terraform Key Test Updated"),
					resource.TestCheckResourceAttr("appwrite_project_key.test", "scopes.#", "2"),
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
