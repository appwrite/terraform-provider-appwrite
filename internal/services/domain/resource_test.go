package domain_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
)

// domainPreCheck gates the domain tests. Domains are served only by Appwrite
// Cloud, and adding one needs a real public domain that no other organization
// has added, so the test cannot make one up: it is named by
// APPWRITE_TEST_DOMAIN. The domain is added and removed again, which deletes
// any zone Appwrite already holds for it, so use one set aside for testing.
func domainPreCheck(t *testing.T) {
	t.Helper()
	acceptance.CloudPreCheck(t)
	acceptance.OrganizationPreCheck(t)
	if os.Getenv("APPWRITE_TEST_DOMAIN") == "" {
		t.Skip("APPWRITE_TEST_DOMAIN must name a public domain set aside for testing to run domain acceptance tests")
	}
}

func TestAccDomain_records(t *testing.T) {
	domain := os.Getenv("APPWRITE_TEST_DOMAIN")
	organizationID := os.Getenv("APPWRITE_ORGANIZATION_ID")

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { domainPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainConfig(domain, "203.0.113.10", 3600, `comment = "SPF"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_domain.test", "domain", domain),
					resource.TestCheckResourceAttr("appwrite_domain.test", "organization_id", organizationID),
					resource.TestCheckResourceAttrSet("appwrite_domain.test", "appwrite_nameservers.0"),
					resource.TestCheckResourceAttrSet("appwrite_domain.test", "created_at"),

					resource.TestCheckResourceAttr("appwrite_domain_record.a", "domain", domain),
					resource.TestCheckResourceAttr("appwrite_domain_record.a", "locked", "false"),
					resource.TestCheckNoResourceAttr("appwrite_domain_record.a", "priority"),
					resource.TestCheckNoResourceAttr("appwrite_domain_record.a", "comment"),

					// Written the long way in configuration; kept as written
					// although the server stores "blog" and the lowercase target.
					resource.TestCheckResourceAttr("appwrite_domain_record.cname", "name", "Blog."+domain+"."),
					resource.TestCheckResourceAttr("appwrite_domain_record.cname", "value", "Target.Example.org."),

					resource.TestCheckResourceAttr("appwrite_domain_record.mx", "priority", "10"),
					resource.TestCheckNoResourceAttr("appwrite_domain_record.mx", "weight"),

					resource.TestCheckResourceAttr("appwrite_domain_record.srv", "priority", "10"),
					resource.TestCheckResourceAttr("appwrite_domain_record.srv", "weight", "5"),
					resource.TestCheckResourceAttr("appwrite_domain_record.srv", "port", "5060"),

					resource.TestCheckResourceAttr("appwrite_domain_record.txt", "comment", "SPF"),

					resource.TestCheckResourceAttr("appwrite_domain_record.aaaa", "value", "2001:db8::10"),
					resource.TestCheckResourceAttr("appwrite_domain_record.alias", "value", "target.example.org"),
					resource.TestCheckResourceAttr("appwrite_domain_record.caa", "value", `0 iodef "mailto:security@example.org"`),
					resource.TestCheckResourceAttr("appwrite_domain_record.https", "value", "1 . alpn=h2"),
					resource.TestCheckResourceAttr("appwrite_domain_record.ns", "value", "ns1.example.org"),

					resource.TestCheckResourceAttrPair("data.appwrite_domain.test", "id", "appwrite_domain.test", "id"),
					resource.TestCheckResourceAttrPair("data.appwrite_domain.test", "appwrite_nameservers.#", "appwrite_domain.test", "appwrite_nameservers.#"),
				),
			},
			{
				ResourceName:      "appwrite_domain.test",
				ImportState:       true,
				ImportStateIdFunc: importID("appwrite_domain.test", organizationID, "id"),
				ImportStateVerify: true,
			},
			{
				ResourceName:      "appwrite_domain_record.mx",
				ImportState:       true,
				ImportStateIdFunc: importID("appwrite_domain_record.mx", "", "domain_id", "id"),
				ImportStateVerify: true,
			},
			{
				// Changes in place: the value and TTL, and the comment removed,
				// which the API only clears when an empty one is sent.
				Config: testAccDomainConfig(domain, "203.0.113.11", 600, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_domain_record.a", "value", "203.0.113.11"),
					resource.TestCheckResourceAttr("appwrite_domain_record.a", "ttl", "600"),
					// The NS record shares the TTL, so the update route of a
					// second type is exercised too.
					resource.TestCheckResourceAttr("appwrite_domain_record.ns", "ttl", "600"),
					resource.TestCheckNoResourceAttr("appwrite_domain_record.txt", "comment"),
				),
			},
			acceptance.ExpectEmptyPlanAfterApply(testAccDomainConfig(domain, "203.0.113.11", 600, "")),
		},
	})
}

// importID builds an import ID from attributes in state, joined by "/" and
// optionally prefixed with a literal.
func importID(resourceName, prefix string, attrs ...string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("%s not found in state", resourceName)
		}
		id := prefix
		for _, attr := range attrs {
			if id != "" {
				id += "/"
			}
			id += rs.Primary.Attributes[attr]
		}
		return id, nil
	}
}

func testAccDomainConfig(domain, address string, ttl int, txtComment string) string {
	return fmt.Sprintf(`
resource "appwrite_domain" "test" {
  domain = %[1]q
}

resource "appwrite_domain_record" "a" {
  domain_id = appwrite_domain.test.id
  type      = "A"
  name      = "www"
  value     = %[2]q
  ttl       = %[3]d
}

resource "appwrite_domain_record" "cname" {
  domain_id = appwrite_domain.test.id
  type      = "CNAME"
  name      = "Blog.%[1]s."
  value     = "Target.Example.org."
}

# One of each remaining type, so every create and update route reaches the
# server at least once.
resource "appwrite_domain_record" "aaaa" {
  domain_id = appwrite_domain.test.id
  type      = "AAAA"
  name      = "www"
  value     = "2001:db8::10"
}

resource "appwrite_domain_record" "alias" {
  domain_id = appwrite_domain.test.id
  type      = "ALIAS"
  name      = "edge"
  value     = "target.example.org"
}

# Appwrite holds its own locked "issue" records at the apex, so an iodef
# record cannot duplicate one of them.
resource "appwrite_domain_record" "caa" {
  domain_id = appwrite_domain.test.id
  type      = "CAA"
  name      = "@"
  value     = "0 iodef \"mailto:security@example.org\""
}

resource "appwrite_domain_record" "https" {
  domain_id = appwrite_domain.test.id
  type      = "HTTPS"
  name      = "svc"
  value     = "1 . alpn=h2"
}

resource "appwrite_domain_record" "ns" {
  domain_id = appwrite_domain.test.id
  type      = "NS"
  name      = "delegated"
  value     = "ns1.example.org"
  ttl       = %[3]d
}

resource "appwrite_domain_record" "mx" {
  domain_id = appwrite_domain.test.id
  type      = "MX"
  name      = "@"
  value     = "mail.example.org"
  priority  = 10
}

resource "appwrite_domain_record" "srv" {
  domain_id = appwrite_domain.test.id
  type      = "SRV"
  name      = "_sip._tcp"
  value     = "sip.example.org"
  priority  = 10
  weight    = 5
  port      = 5060
}

resource "appwrite_domain_record" "txt" {
  domain_id = appwrite_domain.test.id
  type      = "TXT"
  name      = "@"
  value     = "v=spf1 -all"
  %[4]s
}

data "appwrite_domain" "test" {
  domain = appwrite_domain.test.domain
}
`, domain, address, ttl, txtComment)
}

// TestAccDomainNameservers_appwrite sets a domain registered through Appwrite
// back to Appwrite's own nameservers, which is the one change that is safe to
// make to a real registration. Named by APPWRITE_TEST_REGISTERED_DOMAIN_ID,
// since a registration cannot be bought by a test.
func TestAccDomainNameservers_appwrite(t *testing.T) {
	domainID := os.Getenv("APPWRITE_TEST_REGISTERED_DOMAIN_ID")

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck: func() {
			acceptance.CloudPreCheck(t)
			acceptance.OrganizationPreCheck(t)
			if domainID == "" {
				t.Skip("APPWRITE_TEST_REGISTERED_DOMAIN_ID must name a domain registered through Appwrite")
			}
		},
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "appwrite_domain_nameservers" "test" {
  domain_id = %q
}

data "appwrite_domain" "test" {
  id = appwrite_domain_nameservers.test.id
}
`, domainID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_domain_nameservers.test", "id", domainID),
					resource.TestCheckNoResourceAttr("appwrite_domain_nameservers.test", "nameservers"),
					resource.TestCheckResourceAttr("data.appwrite_domain.test", "nameservers_verified", "true"),
				),
			},
		},
	})
}

// TestAccDomainNameserversVerification checks a domain already delegated to
// Appwrite, named by APPWRITE_TEST_DELEGATED_DOMAIN_ID. Delegating one is a
// registrar change no test can make.
func TestAccDomainNameserversVerification(t *testing.T) {
	domainID := os.Getenv("APPWRITE_TEST_DELEGATED_DOMAIN_ID")

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck: func() {
			acceptance.CloudPreCheck(t)
			acceptance.OrganizationPreCheck(t)
			if domainID == "" {
				t.Skip("APPWRITE_TEST_DELEGATED_DOMAIN_ID must name a domain delegated to Appwrite's nameservers")
			}
		},
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "appwrite_domain_nameservers_verification" "test" {
  domain_id = %q

  timeouts {
    create = "2m"
  }
}
`, domainID),
				Check: resource.TestCheckResourceAttr("appwrite_domain_nameservers_verification.test", "id", domainID),
			},
		},
	})
}
