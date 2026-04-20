package user_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_auth_user" "test" {
  email    = "ds-test@example.com"
  password = "password123456"
  name     = "DS Test User"
}

data "appwrite_auth_user" "test" {
  id = appwrite_auth_user.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.appwrite_auth_user.test", "name", "DS Test User"),
					resource.TestCheckResourceAttr("data.appwrite_auth_user.test", "email", "ds-test@example.com"),
					resource.TestCheckResourceAttr("data.appwrite_auth_user.test", "status", "true"),
					resource.TestCheckResourceAttrSet("data.appwrite_auth_user.test", "created_at"),
				),
			},
		},
	})
}
