package function_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFunctionDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_function" "test_ds" {
  name    = "DS Test Function"
  runtime = "node-22"
}

data "appwrite_function" "test" {
  id = appwrite_function.test_ds.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.appwrite_function.test", "name", "DS Test Function"),
					resource.TestCheckResourceAttr("data.appwrite_function.test", "runtime", "node-22"),
					resource.TestCheckResourceAttr("data.appwrite_function.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("data.appwrite_function.test", "created_at"),
				),
			},
		},
	})
}
