package waf_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
)

// The WAF API is served only by Appwrite Cloud; self-hosted Appwrite answers
// its routes with a 404 page. These tests stay off unless APPWRITE_CLOUD_TESTS
// is set, and need a project key with the wafRules.read and wafRules.write
// scopes.
//
// Every rule here is enforced against the test project's real traffic, which
// includes the provider's own requests, so each one matches only a TEST-NET
// address or a path nothing calls. No step holds more than two rules, so
// the tests fit a plan with a small per-project rule limit; they do not run
// in parallel.

// testNetIP is reserved for documentation (RFC 5737), so no request comes
// from it.
const testNetIP = `jsonencode({ method = "equal", attribute = "ip", values = ["192.0.2.10"] })`

const unusedPath = `jsonencode({ method = "startsWith", attribute = "path", values = ["/v1/tf-acc-test-unused"] })`

func TestAccWAFRule_denyAndBypass(t *testing.T) {
	// Rule IDs are unique across Appwrite Cloud, not only within a project.
	bypassID := fmt.Sprintf("tf-acc-test-waf-%d", time.Now().UnixNano())

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.CloudPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDenyAndBypassConfig(bypassID, "deny", 10, true, `description = "Blocks a documentation address"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_waf_rule.deny", "id"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.deny", "type", "deny"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.deny", "resource_type", "api"),
					resource.TestCheckNoResourceAttr("appwrite_waf_rule.deny", "resource_id"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.deny", "priority", "10"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.deny", "enabled", "true"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.deny", "conditions.#", "1"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.deny", "description", "Blocks a documentation address"),
					resource.TestCheckNoResourceAttr("appwrite_waf_rule.deny", "limit"),
					resource.TestCheckNoResourceAttr("appwrite_waf_rule.deny", "difficulty"),
					resource.TestCheckResourceAttrSet("appwrite_waf_rule.deny", "created_at"),

					resource.TestCheckResourceAttr("appwrite_waf_rule.bypass", "id", bypassID),
					resource.TestCheckResourceAttr("appwrite_waf_rule.bypass", "type", "bypass"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.bypass", "priority", "0"),
					resource.TestCheckNoResourceAttr("appwrite_waf_rule.bypass", "description"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.bypass", "conditions.#", "2"),
				),
			},
			{
				ResourceName:      "appwrite_waf_rule.deny",
				ImportState:       true,
				ImportStateIdFunc: importID("appwrite_waf_rule.deny"),
				ImportStateVerify: true,
			},
			{
				ResourceName:      "appwrite_waf_rule.bypass",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Renamed, reprioritized, disabled, description removed:
				// all in place.
				Config: testAccDenyAndBypassConfig(bypassID, "deny renamed", -10, false, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("appwrite_waf_rule.deny", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_waf_rule.deny", "name", "tf-acc-test deny renamed"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.deny", "priority", "-10"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.deny", "enabled", "false"),
					resource.TestCheckNoResourceAttr("appwrite_waf_rule.deny", "description"),
				),
			},
		},
	})
}

func testAccDenyAndBypassConfig(bypassID, name string, priority int, enabled bool, extra string) string {
	return fmt.Sprintf(`
resource "appwrite_waf_rule" "deny" {
  type       = "deny"
  name       = "tf-acc-test %s"
  priority   = %d
  enabled    = %t
  conditions = [%s]
  %s
}

resource "appwrite_waf_rule" "bypass" {
  id   = %q
  type = "bypass"
  name = "tf-acc-test bypass"
  conditions = [
    %s,
    jsonencode({
      method = "or"
      values = [
        { method = "equal", attribute = "country", values = ["AQ"] },
        { method = "equal", attribute = "method", values = ["TRACE"] },
      ]
    }),
  ]
}
`, name, priority, enabled, testNetIP, extra, bypassID, testNetIP)
}

func TestAccWAFRule_challenge(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.CloudPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccChallengeConfig(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_waf_rule.test", "challenge_type", "compute"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.test", "difficulty", "3"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.test", "ttl", "1800"),
				),
			},
			{
				ResourceName:      "appwrite_waf_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccChallengeConfig("difficulty = 5\n  ttl = 900"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_waf_rule.test", "difficulty", "5"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.test", "ttl", "900"),
				),
			},
			{
				// Removing the settings plans them back to their defaults.
				Config: testAccChallengeConfig(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_waf_rule.test", "difficulty", "3"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.test", "ttl", "1800"),
				),
			},
		},
	})
}

func testAccChallengeConfig(extra string) string {
	return fmt.Sprintf(`
resource "appwrite_waf_rule" "test" {
  type       = "challenge"
  name       = "tf-acc-test challenge"
  conditions = [%s]
  %s
}
`, unusedPath, extra)
}

func TestAccWAFRule_rateLimit(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.CloudPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRateLimitConfig(100, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_waf_rule.fixed", "strategy", "fixedWindow"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.fixed", "key", "ip"),
					resource.TestCheckNoResourceAttr("appwrite_waf_rule.fixed", "max_bucket_size"),

					resource.TestCheckResourceAttr("appwrite_waf_rule.bucket", "strategy", "tokenBucket"),
					// Defaults to the limit.
					resource.TestCheckResourceAttr("appwrite_waf_rule.bucket", "max_bucket_size", "100"),
				),
			},
			{
				// Reading a rule back reports the strategy and bucket size
				// only in its configuration; an import proves they are read
				// from there.
				ResourceName:      "appwrite_waf_rule.bucket",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccRateLimitConfig(200, `max_bucket_size = 500`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_waf_rule.fixed", "limit", "200"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.fixed", "key", "userId"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.bucket", "limit", "200"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.bucket", "max_bucket_size", "500"),
				),
			},
		},
	})
}

func testAccRateLimitConfig(limit int, bucketExtra string) string {
	key := `"ip"`
	if limit != 100 {
		key = `"userId"`
	}
	return fmt.Sprintf(`
resource "appwrite_waf_rule" "fixed" {
  type       = "rateLimit"
  name       = "tf-acc-test fixed window"
  limit      = %d
  interval   = 60
  key        = %s
  conditions = [%s]
}

resource "appwrite_waf_rule" "bucket" {
  type       = "rateLimit"
  name       = "tf-acc-test token bucket"
  limit      = %d
  interval   = 60
  strategy   = "tokenBucket"
  conditions = [%s]
  %s
}
`, limit, key, unusedPath, limit, unusedPath, bucketExtra)
}

func TestAccWAFRule_redirect(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.CloudPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedirectConfig("/maintenance", 302),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_waf_rule.test", "location", "/maintenance"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.test", "status_code", "302"),
				),
			},
			{
				ResourceName:      "appwrite_waf_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccRedirectConfig("https://example.com/moved", 308),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_waf_rule.test", "location", "https://example.com/moved"),
					resource.TestCheckResourceAttr("appwrite_waf_rule.test", "status_code", "308"),
				),
			},
		},
	})
}

func testAccRedirectConfig(location string, status int) string {
	return fmt.Sprintf(`
resource "appwrite_waf_rule" "test" {
  type        = "redirect"
  name        = "tf-acc-test redirect"
  location    = %q
  status_code = %d
  conditions  = [%s]
}
`, location, status, unusedPath)
}

// TestAccWAFRule_replacements covers the two changes the API cannot make in
// place: a rate limit rule's strategy, and taking away every condition.
func TestAccWAFRule_replacements(t *testing.T) {
	acceptance.ResourceTestAllowingReplace(t, resource.TestCase{
		PreCheck:                 func() { acceptance.CloudPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReplacementConfig("fixedWindow", true),
			},
			{
				Config: testAccReplacementConfig("slidingWindow", true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("appwrite_waf_rule.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.TestCheckResourceAttr("appwrite_waf_rule.test", "strategy", "slidingWindow"),
			},
			{
				// A disabled rule, so a rule matching every request does not
				// rate limit the test's own requests.
				Config: testAccReplacementConfig("slidingWindow", false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("appwrite_waf_rule.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.TestCheckNoResourceAttr("appwrite_waf_rule.test", "conditions"),
			},
		},
	})
}

func testAccReplacementConfig(strategy string, withConditions bool) string {
	conditions := `enabled = false`
	if withConditions {
		conditions = fmt.Sprintf("conditions = [%s]", unusedPath)
	}
	return fmt.Sprintf(`
resource "appwrite_waf_rule" "test" {
  type     = "rateLimit"
  name     = "tf-acc-test replacement"
  limit    = 10
  interval = 60
  strategy = %q
  %s
}
`, strategy, conditions)
}

// importID builds a project_id/rule_id import ID, exercising the two-part
// form the other import steps do not.
func importID(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}
		return rs.Primary.Attributes["project_id"] + "/" + rs.Primary.ID, nil
	}
}
