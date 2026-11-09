package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccMessagingProviderResource_sendgrid(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMessagingProviderSendgrid("sendgrid-test", "Sendgrid"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "id", "sendgrid-test"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "name", "Sendgrid"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "type", "sendgrid"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("appwrite_messaging_provider.test", "created_at"),
				),
			},
			{
				ResourceName:            "appwrite_messaging_provider.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"api_key"},
			},
			{
				Config: testAccMessagingProviderSendgrid("sendgrid-test", "Sendgrid Updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "name", "Sendgrid Updated"),
				),
			},
		},
	})
}

func TestAccMessagingProviderResource_smtp(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_messaging_provider" "test" {
  id         = "smtp-test"
  name       = "SMTP"
  type       = "smtp"
  host       = "smtp.example.com"
  port       = 587
  encryption = "tls"
  from_email = "noreply@example.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "type", "smtp"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "enabled", "true"),
				),
			},
		},
	})
}

func TestAccMessagingProviderResource_twilio(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_messaging_provider" "test" {
  id   = "twilio-test"
  name = "Twilio"
  type = "twilio"
  account_sid = "ACtestaccountsid"
  auth_token  = "testauthtoken"
  from = "+1234567890"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "type", "twilio"),
				),
			},
		},
	})
}

func TestAccMessagingProviderResource_ses(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMessagingProviderSES("SES", "us-east-1", "Test App"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "id", "tf-acc-test-ses"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "name", "SES"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "type", "ses"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "enabled", "true"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "region", "us-east-1"),
					resource.TestCheckResourceAttrSet("appwrite_messaging_provider.test", "created_at"),
					testAccCheckProviderOnServer(t, "tf-acc-test-ses", "ses",
						map[string]any{"accessKey": "AKIAIOSFODNN7EXAMPLE", "region": "us-east-1"},
						map[string]any{"fromEmail": "noreply@example.com", "fromName": "Test App"}),
				),
			},
			{
				ResourceName:      "appwrite_messaging_provider.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Credentials are not read back, as for every other type.
				ImportStateVerifyIgnore: []string{"access_key", "secret_key", "region"},
			},
			{
				Config: testAccMessagingProviderSES("SES Updated", "eu-west-1", "Test App Updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "name", "SES Updated"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "region", "eu-west-1"),
					testAccCheckProviderOnServer(t, "tf-acc-test-ses", "ses",
						map[string]any{"region": "eu-west-1"},
						map[string]any{"fromName": "Test App Updated"}),
				),
			},
		},
	})
}

func TestAccMessagingProviderResource_appwrite(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		// The appwrite push provider arrived in Appwrite 2.3.0.
		PreCheck:                 func() { acceptance.ServerVersionPreCheck(t, "2.3.0") },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_messaging_provider" "test" {
  id     = "tf-acc-test-appwrite-push"
  name   = "Appwrite Push"
  type   = "appwrite"
  qos    = 1
  expiry = 3600
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "type", "appwrite"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "enabled", "true"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "qos", "1"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "expiry", "3600"),
					testAccCheckProviderOnServer(t, "tf-acc-test-appwrite-push", "appwrite", nil,
						map[string]any{"qos": float64(1), "expiry": float64(3600)}),
				),
			},
			{
				ResourceName:      "appwrite_messaging_provider.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Removing qos clears it on the server rather than leaving the
				// old value in place.
				Config: `
resource "appwrite_messaging_provider" "test" {
  id      = "tf-acc-test-appwrite-push"
  name    = "Appwrite Push"
  type    = "appwrite"
  enabled = false
  expiry  = 60
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("appwrite_messaging_provider.test", "qos"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "expiry", "60"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "enabled", "false"),
					testAccCheckProviderOnServer(t, "tf-acc-test-appwrite-push", "appwrite", nil,
						map[string]any{"qos": nil, "expiry": float64(60)}),
				),
			},
		},
	})
}

// TestAccMessagingProviderResource_appwriteUnsupported checks that an Appwrite
// without the appwrite provider route, such as the version CI runs, fails with
// an explanation rather than its HTML 404 page.
func TestAccMessagingProviderResource_appwriteUnsupported(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck: func() {
			acceptance.PreCheck(t)
			if version, ok := acceptance.ServerVersionAtLeast(t, "2.3.0"); ok {
				t.Skipf("Appwrite %s serves the appwrite provider type", version)
			}
		},
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_messaging_provider" "test" {
  name = "Appwrite Push"
  type = "appwrite"
}
`,
				ExpectError: regexp.MustCompile(`which\s+Appwrite\s+2\.3\.0\s+added`),
			},
		},
	})
}

func TestAccMessagingProviderResource_argumentsForOtherType(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_messaging_provider" "test" {
  name    = "Sendgrid"
  type    = "sendgrid"
  api_key = "SG.test-key"
  region  = "us-east-1"
}
`,
				ExpectError: regexp.MustCompile(`region\s+applies\s+only\s+to\s+ses\s+providers,\s+not\s+sendgrid`),
			},
			{
				Config: `
resource "appwrite_messaging_provider" "test" {
  name = "FCM"
  type = "fcm"
  qos  = 1
}
`,
				ExpectError: regexp.MustCompile(`qos\s+applies\s+only\s+to\s+appwrite\s+providers,\s+not\s+fcm`),
			},
		},
	})
}

func testAccMessagingProviderSES(name, region, fromName string) string {
	return fmt.Sprintf(`
resource "appwrite_messaging_provider" "test" {
  id         = "tf-acc-test-ses"
  name       = %q
  type       = "ses"
  access_key = "AKIAIOSFODNN7EXAMPLE"
  secret_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  region     = %q
  from_email = "noreply@example.com"
  from_name  = %q
}
`, name, region, fromName)
}

// testAccCheckProviderOnServer reads the provider back from Appwrite and
// checks its service and the given credentials and options. A nil value
// expects the key to be present and null.
func testAccCheckProviderOnServer(t *testing.T, id, service string, credentials, options map[string]any) resource.TestCheckFunc {
	return func(*terraform.State) error {
		prov, err := appwrite.NewMessaging(acceptance.ProjectClient(t)).GetProvider(id)
		if err != nil {
			return fmt.Errorf("reading provider %s: %w", id, err)
		}
		if prov.Provider != service {
			return fmt.Errorf("provider %s is a %s provider, want %s", id, prov.Provider, service)
		}
		for _, check := range []struct {
			name string
			got  any
			want map[string]any
		}{
			{"credentials", prov.Credentials, credentials},
			{"options", prov.Options, options},
		} {
			got, _ := check.got.(map[string]any)
			for key, want := range check.want {
				actual, ok := got[key]
				if !ok {
					return fmt.Errorf("provider %s %s has no %s", id, check.name, key)
				}
				if actual != want {
					return fmt.Errorf("provider %s %s.%s = %v, want %v", id, check.name, key, actual, want)
				}
			}
		}
		return nil
	}
}

func testAccMessagingProviderSendgrid(id, name string) string {
	return fmt.Sprintf(`
resource "appwrite_messaging_provider" "test" {
  id         = %q
  name       = %q
  type       = "sendgrid"
  api_key    = "SG.test-key"
  from_email = "noreply@example.com"
  from_name  = "Test App"
}
`, id, name)
}
