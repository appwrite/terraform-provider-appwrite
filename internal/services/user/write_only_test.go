package user_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// The point of a write-only argument is what is absent: the password reaches
// Appwrite but never the state file. ImportStateVerify would not catch a
// regression here, because a write-only attribute is absent from state by
// definition -- so this asserts on the raw state instead.
func TestAccUserResource_writeOnlyPassword(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() { acceptance.PreCheck(t) },
		// Write-only arguments arrived in Terraform 1.11.
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserWriteOnlyConfig("S3cret-one!", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_auth_user.wo", "password_wo_version", "1"),
					resource.TestCheckNoResourceAttr("appwrite_auth_user.wo", "password_wo"),
					resource.TestCheckNoResourceAttr("appwrite_auth_user.wo", "password"),
				),
			},
			{
				// Bumping the version is the only way to signal a new secret,
				// since Terraform cannot diff a value it never stored.
				Config: testAccUserWriteOnlyConfig("S3cret-two!", 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_auth_user.wo", "password_wo_version", "2"),
					resource.TestCheckNoResourceAttr("appwrite_auth_user.wo", "password_wo"),
				),
			},
		},
	})
}

func testAccUserWriteOnlyConfig(password string, version int) string {
	return fmt.Sprintf(`
resource "appwrite_auth_user" "wo" {
  id                  = "wo-user"
  email               = "wo-user@example.com"
  password_wo         = %q
  password_wo_version = %d
}
`, password, version)
}

// Moving an existing user from `password` to `password_wo` must actually change
// the password. Before the version became mandatory, both versions were null on
// that transition, nothing compared unequal, and the apply reported success
// while Appwrite kept the old password.
func TestAccUserResource_migrateToWriteOnlyPassword(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() { acceptance.PreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_auth_user" "migrate" {
  id       = "wo-migrate"
  email    = "wo-migrate@example.com"
  password = "S3cret-stored!"
}
`,
			},
			{
				// Omitting password_wo_version here is rejected by the schema,
				// which is what stops the silent no-op.
				Config: `
resource "appwrite_auth_user" "migrate" {
  id          = "wo-migrate"
  email       = "wo-migrate@example.com"
  password_wo = "S3cret-writeonly!"
}
`,
				ExpectError: regexp.MustCompile(`password_wo_version`),
			},
			{
				Config: `
resource "appwrite_auth_user" "migrate" {
  id                  = "wo-migrate"
  email               = "wo-migrate@example.com"
  password_wo         = "S3cret-writeonly!"
  password_wo_version = 1
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_auth_user.migrate", "password_wo_version", "1"),
					resource.TestCheckNoResourceAttr("appwrite_auth_user.migrate", "password"),
				),
			},
		},
	})
}
