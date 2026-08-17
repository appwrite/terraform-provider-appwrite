package project_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProjectResource_basic(t *testing.T) {
	projectID := fmt.Sprintf("tf-%d", time.Now().Unix())
	organizationID := os.Getenv("APPWRITE_ORGANIZATION_ID")
	region := os.Getenv("APPWRITE_TEST_REGION")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.OrganizationPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfig(projectID, "Terraform Project Test", region),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_project.test", "id", projectID),
					resource.TestCheckResourceAttr("appwrite_project.test", "name", "Terraform Project Test"),
					resource.TestCheckResourceAttr("appwrite_project.test", "organization_id", organizationID),
					resource.TestCheckResourceAttrSet("appwrite_project.test", "region"),
					resource.TestCheckResourceAttrSet("appwrite_project.test", "created_at"),
				),
			},
			{
				ResourceName:      "appwrite_project.test",
				ImportState:       true,
				ImportStateId:     organizationID + "/" + projectID,
				ImportStateVerify: true,
			},
			{
				Config: testAccProjectConfig(projectID, "Terraform Project Test Updated", region),
				Check: resource.TestCheckResourceAttr(
					"appwrite_project.test",
					"name",
					"Terraform Project Test Updated",
				),
			},
		},
	})
}

func testAccProjectConfig(projectID, name, region string) string {
	regionAttribute := ""
	if region != "" {
		regionAttribute = fmt.Sprintf("\n  region = %q", region)
	}
	return fmt.Sprintf(`
resource "appwrite_project" "test" {
  id   = %q
  name = %q%s
}
`, projectID, name, regionAttribute)
}
