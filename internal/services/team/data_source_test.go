package team_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTeamDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_auth_team" "test" {
  id   = "ds-test-team"
  name = "DS Test Team"
}

data "appwrite_auth_team" "test" {
  id = appwrite_auth_team.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.appwrite_auth_team.test", "name", "DS Test Team"),
					resource.TestCheckResourceAttrSet("data.appwrite_auth_team.test", "created_at"),
				),
			},
		},
	})
}
