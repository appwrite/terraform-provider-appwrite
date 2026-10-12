package function_test

import (
	"regexp"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFunctionResource_basic(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
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
	acceptance.ResourceTest(t, resource.TestCase{
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

// Intervals reached Appwrite Cloud before a self-hosted release, and CI runs self-hosted.
func TestAccFunctionResource_interval(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.CloudPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_function" "test_interval" {
  name     = "Interval Function"
  runtime  = "node-22"
  interval = 60
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_function.test_interval", "interval", "60"),
					resource.TestCheckNoResourceAttr("appwrite_function.test_interval", "schedule"),
				),
			},
			{
				ResourceName:      "appwrite_function.test_interval",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: `
resource "appwrite_function" "test_interval" {
  name     = "Interval Function"
  runtime  = "node-22"
  schedule = "0 * * * *"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_function.test_interval", "schedule", "0 * * * *"),
					resource.TestCheckNoResourceAttr("appwrite_function.test_interval", "interval"),
				),
			},
			{
				Config: `
resource "appwrite_function" "test_interval" {
  name     = "Interval Function"
  runtime  = "node-22"
  interval = 120
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_function.test_interval", "interval", "120"),
					resource.TestCheckNoResourceAttr("appwrite_function.test_interval", "schedule"),
				),
			},
			{
				Config: `
resource "appwrite_function" "test_interval" {
  name    = "Interval Function"
  runtime = "node-22"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("appwrite_function.test_interval", "interval"),
				),
			},
			{
				Config: `
resource "appwrite_function" "test_interval" {
  name     = "Interval Function"
  runtime  = "node-22"
  interval = 60
  schedule = "0 * * * *"
}
`,
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}
