package monitor_test

import (
	"regexp"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Probe monitors are served only by Appwrite Cloud, and the self-hosted
// release CI runs predates the monitors API altogether. These tests stay off
// unless APPWRITE_CLOUD_TESTS is set.
func TestAccMonitorResource_http(t *testing.T) {
	const name = "appwrite_monitor.test"

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.CloudPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_monitor" "test" {
  type = "http"
  name = "tf-acc-test-monitor"
  url  = "https://example.com"
  port = 443
}
`,
				ExpectError: regexp.MustCompile(`port is used only by tcp and smtp monitors`),
			},
			{
				Config: `
resource "appwrite_monitor" "test" {
  type = "http"
  name = "tf-acc-test-monitor"
  url  = "https://example.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(name, "id"),
					resource.TestCheckResourceAttr(name, "method", "GET"),
					resource.TestCheckResourceAttr(name, "timeout", "10"),
					resource.TestCheckResourceAttrSet(name, "interval"),
					resource.TestCheckResourceAttr(name, "cooldown", "3600"),
					resource.TestCheckResourceAttr(name, "enabled", "true"),
					resource.TestCheckResourceAttr(name, "severity", "warning"),
					resource.TestCheckResourceAttr(name, "notify.#", "1"),
					resource.TestCheckResourceAttr(name, "notify.0", "owner"),
					resource.TestCheckResourceAttr(name, "escalation_id", ""),
					resource.TestCheckNoResourceAttr(name, "rules"),
					resource.TestCheckNoResourceAttr(name, "headers"),
					resource.TestCheckNoResourceAttr(name, "body"),
					resource.TestCheckNoResourceAttr(name, "host"),
					resource.TestCheckNoResourceAttr(name, "window"),
				),
			},
			{
				ResourceName:            name,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"rules", "status"},
			},
			{
				Config: `
resource "appwrite_monitor" "test" {
  type     = "http"
  name     = "tf-acc-test-monitor-updated"
  url      = "https://example.com/health"
  method   = "POST"
  body     = "{}"
  timeout  = 5
  interval = 300
  enabled  = false
  severity = "critical"
  notify   = []

  headers = {
    "X-Check" = "terraform"
  }

  rules = [
    jsonencode({ values = [2000], attribute = "latency", method = "greaterThan" }),
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "tf-acc-test-monitor-updated"),
					resource.TestCheckResourceAttr(name, "url", "https://example.com/health"),
					resource.TestCheckResourceAttr(name, "method", "POST"),
					resource.TestCheckResourceAttr(name, "body", "{}"),
					resource.TestCheckResourceAttr(name, "timeout", "5"),
					resource.TestCheckResourceAttr(name, "interval", "300"),
					resource.TestCheckResourceAttr(name, "enabled", "false"),
					resource.TestCheckResourceAttr(name, "severity", "critical"),
					resource.TestCheckResourceAttr(name, "notify.#", "0"),
					resource.TestCheckResourceAttr(name, "headers.X-Check", "terraform"),
					resource.TestCheckResourceAttr(name, "rules.#", "1"),
				),
			},
			{
				// Leaving the optional arguments out puts each back to its
				// default, which is what clears the body and rules. Headers
				// stay: removing all of them replaces the monitor.
				Config: `
resource "appwrite_monitor" "test" {
  type     = "http"
  name     = "tf-acc-test-monitor"
  url      = "https://example.com"
  interval = 300

  headers = {
    "X-Check" = "terraform"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "method", "GET"),
					resource.TestCheckResourceAttr(name, "timeout", "10"),
					resource.TestCheckResourceAttr(name, "enabled", "true"),
					resource.TestCheckResourceAttr(name, "notify.0", "owner"),
					resource.TestCheckResourceAttr(name, "headers.X-Check", "terraform"),
					resource.TestCheckNoResourceAttr(name, "body"),
					resource.TestCheckNoResourceAttr(name, "rules"),
				),
			},
		},
	})
}

func TestAccMonitorResource_usage(t *testing.T) {
	const name = "appwrite_monitor.test"

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.CloudPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_monitor" "test" {
  type = "usage"
  name = "tf-acc-test-usage"
}
`,
				ExpectError: regexp.MustCompile(`usage monitors require rules`),
			},
			{
				// The scope rules come first here; Appwrite stores the metric
				// rule first, which must not show as a diff.
				Config: `
resource "appwrite_monitor" "test" {
  type = "usage"
  name = "tf-acc-test-usage"

  rules = [
    jsonencode({ method = "equal", attribute = "resourceType", values = ["project"] }),
    jsonencode({ method = "greaterThan", attribute = "network.outbound", values = [5000000000] }),
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "type", "usage"),
					resource.TestCheckResourceAttr(name, "window", "3600"),
					resource.TestCheckResourceAttr(name, "rules.#", "2"),
					resource.TestCheckNoResourceAttr(name, "interval"),
					resource.TestCheckNoResourceAttr(name, "timeout"),
					resource.TestCheckNoResourceAttr(name, "method"),
				),
			},
			{
				ResourceName:            name,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"rules", "status"},
			},
			{
				Config: `
resource "appwrite_monitor" "test" {
  type   = "usage"
  name   = "tf-acc-test-usage"
  window = 86400

  rules = [
    jsonencode({ method = "greaterThan", attribute = "network.outbound", values = [1000000000] }),
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "window", "86400"),
					resource.TestCheckResourceAttr(name, "rules.#", "1"),
				),
			},
		},
	})
}
