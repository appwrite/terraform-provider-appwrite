package proxy_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccProxyRuleResource_site(t *testing.T) {
	domain := testDomain("tf")
	projectID := os.Getenv("APPWRITE_PROJECT_ID")

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProxySiteRuleConfig(domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_proxy_rule.test", "id"),
					resource.TestCheckResourceAttr("appwrite_proxy_rule.test", "domain", domain),
					resource.TestCheckResourceAttr("appwrite_proxy_rule.test", "type", "site"),
					resource.TestCheckResourceAttrPair("appwrite_proxy_rule.test", "resource_id", "appwrite_site.test", "id"),
					resource.TestCheckResourceAttrSet("appwrite_proxy_rule.test", "status"),
				),
			},
			{
				ResourceName: "appwrite_proxy_rule.test",
				ImportState:  true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rule, ok := state.RootModule().Resources["appwrite_proxy_rule.test"]
					if !ok || rule.Primary == nil {
						return "", fmt.Errorf("proxy rule not found in state")
					}
					return projectID + "/" + rule.Primary.ID, nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccProxyRuleResource_function(t *testing.T) {
	domain := testDomain("tf-function")

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProxyFunctionRuleConfig(domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_proxy_rule.test", "id"),
					resource.TestCheckResourceAttr("appwrite_proxy_rule.test", "domain", domain),
					resource.TestCheckResourceAttr("appwrite_proxy_rule.test", "type", "function"),
					resource.TestCheckResourceAttrPair("appwrite_proxy_rule.test", "resource_id", "appwrite_function.test", "id"),
				),
			},
		},
	})
}

func TestAccProxyRuleResource_api(t *testing.T) {
	domain := testDomain("tf-acc-test-api")
	projectID := os.Getenv("APPWRITE_PROJECT_ID")

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProxyAPIRuleConfig(domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_proxy_rule.test", "id"),
					resource.TestCheckResourceAttr("appwrite_proxy_rule.test", "domain", domain),
					resource.TestCheckResourceAttr("appwrite_proxy_rule.test", "type", "api"),
					resource.TestCheckNoResourceAttr("appwrite_proxy_rule.test", "resource_id"),
					resource.TestCheckNoResourceAttr("appwrite_proxy_rule.test", "redirect_status_code"),
					resource.TestCheckResourceAttrSet("appwrite_proxy_rule.test", "status"),
				),
			},
			{
				ResourceName:      "appwrite_proxy_rule.test",
				ImportState:       true,
				ImportStateIdFunc: importID(projectID),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccProxyRuleResource_redirect(t *testing.T) {
	domain := testDomain("tf-acc-test-redirect")
	projectID := os.Getenv("APPWRITE_PROJECT_ID")

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProxyRedirectRuleConfig(domain, "https://example.org/Landing?from=terraform", 307),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_proxy_rule.test", "id"),
					resource.TestCheckResourceAttr("appwrite_proxy_rule.test", "domain", domain),
					resource.TestCheckResourceAttr("appwrite_proxy_rule.test", "type", "redirect"),
					resource.TestCheckResourceAttrPair("appwrite_proxy_rule.test", "resource_id", "appwrite_function.test", "id"),
					resource.TestCheckResourceAttr("appwrite_proxy_rule.test", "redirect_url", "https://example.org/Landing?from=terraform"),
					resource.TestCheckResourceAttr("appwrite_proxy_rule.test", "redirect_status_code", "307"),
					resource.TestCheckResourceAttr("appwrite_proxy_rule.test", "redirect_resource_type", "function"),
				),
			},
			{
				ResourceName:      "appwrite_proxy_rule.test",
				ImportState:       true,
				ImportStateIdFunc: importID(projectID),
				ImportStateVerify: true,
			},
		},
	})
}

// Appwrite has no update for a rule, so changing a redirect replaces it.
func TestAccProxyRuleResource_redirectReplace(t *testing.T) {
	domain := testDomain("tf-acc-test-redirect-replace")

	acceptance.ResourceTestAllowingReplace(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProxyRedirectRuleConfig(domain, "https://example.org", 301),
				Check:  resource.TestCheckResourceAttr("appwrite_proxy_rule.test", "redirect_status_code", "301"),
			},
			{
				Config: testAccProxyRedirectRuleConfig(domain, "https://example.org", 308),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("appwrite_proxy_rule.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.TestCheckResourceAttr("appwrite_proxy_rule.test", "redirect_status_code", "308"),
			},
		},
	})
}

func TestAccProxyRuleResource_invalidConfig(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_proxy_rule" "test" {
  domain = "tf-acc-test-invalid.example.com"
  type   = "redirect"
}
`,
				ExpectError: regexp.MustCompile(`redirect rules require redirect_url`),
			},
			{
				Config: `
resource "appwrite_proxy_rule" "test" {
  domain      = "tf-acc-test-invalid.example.com"
  type        = "api"
  resource_id = "some-site"
}
`,
				ExpectError: regexp.MustCompile(`resource_id is not used by api rules`),
			},
		},
	})
}

func importID(projectID string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		rule, ok := state.RootModule().Resources["appwrite_proxy_rule.test"]
		if !ok || rule.Primary == nil {
			return "", fmt.Errorf("proxy rule not found in state")
		}
		return projectID + "/" + rule.Primary.ID, nil
	}
}

func testDomain(prefix string) string {
	baseDomain := os.Getenv("APPWRITE_TEST_DOMAIN")
	if baseDomain == "" {
		baseDomain = "example.com"
	}
	return fmt.Sprintf("%s-%d.%s", prefix, time.Now().UnixNano(), baseDomain)
}

func testAccProxySiteRuleConfig(domain string) string {
	return fmt.Sprintf(`
resource "appwrite_site" "test" {
  name          = "terraform-proxy-rule-test"
  framework     = "other"
  build_runtime = "node-22"
}

resource "appwrite_proxy_rule" "test" {
  domain      = %q
  type        = "site"
  resource_id = appwrite_site.test.id
}
`, domain)
}

func testAccProxyFunctionRuleConfig(domain string) string {
	return fmt.Sprintf(`
resource "appwrite_function" "test" {
  name    = "terraform-proxy-rule-test"
  runtime = "node-22"
}

resource "appwrite_proxy_rule" "test" {
  domain      = %q
  type        = "function"
  resource_id = appwrite_function.test.id
}
`, domain)
}

func testAccProxyAPIRuleConfig(domain string) string {
	return fmt.Sprintf(`
resource "appwrite_proxy_rule" "test" {
  domain = %q
  type   = "api"
}
`, domain)
}

func testAccProxyRedirectRuleConfig(domain, url string, statusCode int) string {
	return fmt.Sprintf(`
resource "appwrite_function" "test" {
  name    = "terraform-proxy-redirect-test"
  runtime = "node-22"
}

resource "appwrite_proxy_rule" "test" {
  domain                 = %q
  type                   = "redirect"
  resource_id            = appwrite_function.test.id
  redirect_resource_type = "function"
  redirect_url           = %q
  redirect_status_code   = %d
}
`, domain, url, statusCode)
}
