package function_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFunctionResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_function" "test" {
  name    = "Test Function"
  runtime = "node-22"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_function.test", "id"),
					resource.TestCheckResourceAttr("appwrite_function.test", "name", "Test Function"),
					resource.TestCheckResourceAttr("appwrite_function.test", "runtime", "node-22"),
					resource.TestCheckResourceAttr("appwrite_function.test", "enabled", "true"),
					resource.TestCheckResourceAttr("appwrite_function.test", "logging", "true"),
					resource.TestCheckResourceAttrSet("appwrite_function.test", "created_at"),
				),
			},
			{
				ResourceName:      "appwrite_function.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: `
resource "appwrite_function" "test" {
  name    = "Updated Function"
  runtime = "node-22"
  timeout = 30
  enabled = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_function.test", "name", "Updated Function"),
					resource.TestCheckResourceAttr("appwrite_function.test", "timeout", "30"),
					resource.TestCheckResourceAttr("appwrite_function.test", "enabled", "false"),
				),
			},
		},
	})
}

func TestAccFunctionResource_with_events(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_function" "test_events" {
  name       = "Event Function"
  runtime    = "node-22"
  events     = ["databases.*.collections.*.documents.*.create"]
  entrypoint = "index.js"
  commands   = "npm install"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_function.test_events", "id"),
					resource.TestCheckResourceAttr("appwrite_function.test_events", "events.#", "1"),
					resource.TestCheckResourceAttr("appwrite_function.test_events", "entrypoint", "index.js"),
					resource.TestCheckResourceAttr("appwrite_function.test_events", "commands", "npm install"),
				),
			},
		},
	})
}
