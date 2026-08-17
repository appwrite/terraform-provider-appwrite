package proxy_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProxyRuleResource_site(t *testing.T) {
	domain := fmt.Sprintf("tf-%d.example.com", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
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
				ResourceName:      "appwrite_proxy_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccProxyRuleResource_function(t *testing.T) {
	domain := fmt.Sprintf("tf-function-%d.example.com", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
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
