package team_test

import (
	"fmt"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTeamResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamConfig("engineering", "Engineering"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_team.test", "id", "engineering"),
					resource.TestCheckResourceAttr("appwrite_team.test", "name", "Engineering"),
					resource.TestCheckResourceAttrSet("appwrite_team.test", "created_at"),
				),
			},
			{
				ResourceName:      "appwrite_team.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTeamConfig("engineering", "Engineering Team"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_team.test", "name", "Engineering Team"),
				),
			},
		},
	})
}

func testAccTeamConfig(id, name string) string {
	return fmt.Sprintf(`
resource "appwrite_team" "test" {
  id   = %q
  name = %q
}
`, id, name)
}
