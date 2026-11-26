package escalation_test

import (
	"regexp"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Escalations belong to an organization and need an organization API key,
// which self-hosted Appwrite does not issue. This test stays off unless
// APPWRITE_CLOUD_TESTS and APPWRITE_ORGANIZATION_ID are set.
func TestAccEscalationResource_basic(t *testing.T) {
	const name = "appwrite_escalation.test"

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck: func() {
			acceptance.CloudPreCheck(t)
			acceptance.OrganizationPreCheck(t)
		},
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_escalation" "test" {
  name  = "tf-acc-test-escalation"
  steps = [{ roles = ["owner"] }, { delay = 60, roles = [], user_ids = [] }]
}
`,
				ExpectError: regexp.MustCompile(`Each step needs at least one of roles or user_ids`),
			},
			{
				Config: `
resource "appwrite_escalation" "test" {
  name  = "tf-acc-test-escalation"
  steps = [{ roles = ["owner"] }]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(name, "id"),
					resource.TestCheckResourceAttr(name, "repeat", "0"),
					resource.TestCheckResourceAttr(name, "steps.#", "1"),
					resource.TestCheckResourceAttr(name, "steps.0.delay", "0"),
					resource.TestCheckResourceAttr(name, "steps.0.roles.0", "owner"),
					resource.TestCheckResourceAttr(name, "steps.0.user_ids.#", "0"),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: `
resource "appwrite_escalation" "test" {
  name   = "tf-acc-test-escalation-updated"
  repeat = 2

  steps = [
    { roles = ["owner"] },
    { delay = 900, roles = ["developer", "owner"] },
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "tf-acc-test-escalation-updated"),
					resource.TestCheckResourceAttr(name, "repeat", "2"),
					resource.TestCheckResourceAttr(name, "steps.#", "2"),
					resource.TestCheckResourceAttr(name, "steps.1.delay", "900"),
					resource.TestCheckResourceAttr(name, "steps.1.roles.#", "2"),
				),
			},
			{
				// A monitor that uses the escalation is destroyed before it.
				Config: `
resource "appwrite_escalation" "test" {
  name   = "tf-acc-test-escalation-updated"
  repeat = 2

  steps = [
    { roles = ["owner"] },
    { delay = 900, roles = ["developer", "owner"] },
  ]
}

resource "appwrite_monitor" "test" {
  type          = "tcp"
  name          = "tf-acc-test-escalated"
  host          = "example.com"
  port          = 443
  escalation_id = appwrite_escalation.test.id
}
`,
				Check: resource.TestCheckResourceAttrPair("appwrite_monitor.test", "escalation_id", name, "id"),
			},
		},
	})
}
