package user_test

import (
	"fmt"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfig("test-user", "Test User", "test@example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_user.test", "id", "test-user"),
					resource.TestCheckResourceAttr("appwrite_user.test", "name", "Test User"),
					resource.TestCheckResourceAttr("appwrite_user.test", "email", "test@example.com"),
					resource.TestCheckResourceAttr("appwrite_user.test", "status", "true"),
					resource.TestCheckResourceAttrSet("appwrite_user.test", "created_at"),
				),
			},
			{
				ResourceName:            "appwrite_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			{
				Config: testAccUserConfig("test-user", "Updated User", "updated@example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_user.test", "name", "Updated User"),
					resource.TestCheckResourceAttr("appwrite_user.test", "email", "updated@example.com"),
				),
			},
		},
	})
}

func TestAccUserResource_with_labels(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_user" "test" {
  id       = "admin-user"
  name     = "Admin"
  email    = "admin@example.com"
  password = "securepassword123"
  labels   = ["admin", "staff"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_user.test", "labels.#", "2"),
					resource.TestCheckResourceAttr("appwrite_user.test", "labels.0", "admin"),
					resource.TestCheckResourceAttr("appwrite_user.test", "labels.1", "staff"),
				),
			},
		},
	})
}

func testAccUserConfig(id, name, email string) string {
	return fmt.Sprintf(`
resource "appwrite_user" "test" {
  id       = %q
  name     = %q
  email    = %q
  password = "testpassword123"
}
`, id, name, email)
}
