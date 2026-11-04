package user_test

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
)

// TestAccUserTarget_basic creates one target of each type, subscribes the
// email target to a topic, changes everything that can change in place, and
// imports.
//
// The user is a fixture rather than an appwrite_auth_user, so it outlives the
// configuration: deleting a user deletes its targets, and with the user still
// there the destroy check shows the targets were deleted by this resource.
func TestAccUserTarget_basic(t *testing.T) {
	userID, cleanup := acceptance.User(t)
	defer cleanup()

	// Identifiers are unique across the project, so each run uses its own.
	// Base 36 keeps the IDs built from it within Appwrite's 36 characters.
	nanos := time.Now().UnixNano()
	suffix := strconv.FormatInt(nanos, 36)
	before := targetTestConfig{
		userID:   userID,
		suffix:   suffix,
		email:    fmt.Sprintf("tf-acc-test-%s@example.com", suffix),
		phone:    fmt.Sprintf("+1415%07d", nanos%10_000_000),
		name:     "Work",
		provider: "first",
	}
	after := before
	after.email = fmt.Sprintf("tf-acc-test-%s-updated@example.com", suffix)
	after.phone = fmt.Sprintf("+1628%07d", nanos%10_000_000)
	after.name = "Home"
	after.provider = "second"

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: before.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_user_target.email", "id"),
					resource.TestCheckResourceAttr("appwrite_user_target.email", "user_id", userID),
					resource.TestCheckResourceAttr("appwrite_user_target.email", "provider_type", "email"),
					resource.TestCheckResourceAttr("appwrite_user_target.email", "identifier", before.email),
					resource.TestCheckResourceAttr("appwrite_user_target.email", "name", "Work"),
					resource.TestCheckResourceAttrPair("appwrite_user_target.email", "provider_id", "appwrite_messaging_provider.first", "id"),
					resource.TestCheckResourceAttr("appwrite_user_target.email", "expired", "false"),
					resource.TestCheckResourceAttrSet("appwrite_user_target.email", "created_at"),
					resource.TestCheckResourceAttr("appwrite_user_target.sms", "identifier", before.phone),
					resource.TestCheckNoResourceAttr("appwrite_user_target.sms", "name"),
					resource.TestCheckNoResourceAttr("appwrite_user_target.sms", "provider_id"),
					resource.TestCheckResourceAttr("appwrite_user_target.push", "id", fmt.Sprintf("tf-acc-test-push-%s", suffix)),
					resource.TestCheckResourceAttr("appwrite_user_target.push", "provider_type", "push"),
					resource.TestCheckResourceAttrPair("appwrite_messaging_subscriber.email", "target_id", "appwrite_user_target.email", "id"),
					resource.TestCheckResourceAttrPair("appwrite_messaging_subscriber.push", "target_id", "appwrite_user_target.push", "id"),
				),
			},
			{
				// Identifier, name and provider all change in place. The
				// harness fails the step if anything is replaced, so the
				// subscribers keep pointing at the same targets.
				Config: after.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_user_target.email", "identifier", after.email),
					resource.TestCheckResourceAttr("appwrite_user_target.email", "name", "Home"),
					resource.TestCheckResourceAttrPair("appwrite_user_target.email", "provider_id", "appwrite_messaging_provider.second", "id"),
					resource.TestCheckResourceAttr("appwrite_user_target.sms", "identifier", after.phone),
					resource.TestCheckResourceAttrPair("appwrite_messaging_subscriber.email", "target_id", "appwrite_user_target.email", "id"),
				),
			},
			{
				ResourceName:      "appwrite_user_target.email",
				ImportState:       true,
				ImportStateIdFunc: targetImportID("appwrite_user_target.email"),
				ImportStateVerify: true,
			},
			{
				ResourceName:      "appwrite_user_target.push",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s/tf-acc-test-push-%s", userID, suffix),
				ImportStateVerify: true,
			},
			{
				// The target Appwrite keeps for the user's own email address
				// is refused, since Appwrite manages it alongside the user.
				ResourceName: "appwrite_user_target.email",
				ImportState:  true,
				ImportStateIdFunc: func(*terraform.State) (string, error) {
					user, err := appwrite.NewUsers(acceptance.ProjectClient(t)).Get(userID)
					if err != nil {
						return "", err
					}
					for _, target := range user.Targets {
						if target.ProviderType == "email" && target.Identifier == user.Email {
							return userID + "/" + target.Id, nil
						}
					}
					return "", fmt.Errorf("user %s has no email target of its own", userID)
				},
				ExpectError: regexp.MustCompile(`Cannot import a user's own contact target`),
			},
		},
	})
}

// TestAccUserTarget_removingNameReplaces covers the arguments Appwrite cannot
// clear: removing one from the configuration has to replace the target, or the
// apply would report success and leave the old value in place. It also covers
// a target on a user Terraform manages, and the subscriber following the
// replacement to the target's new ID.
func TestAccUserTarget_removingNameReplaces(t *testing.T) {
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	config := func(name string) string {
		nameLine := ""
		if name != "" {
			nameLine = fmt.Sprintf("name = %q", name)
		}
		return fmt.Sprintf(`
resource "appwrite_auth_user" "test" {
  id    = "tf-acc-test-target-%[1]s"
  email = "tf-acc-test-target-%[1]s@example.com"
}

resource "appwrite_user_target" "test" {
  user_id       = appwrite_auth_user.test.id
  provider_type = "email"
  identifier    = "tf-acc-test-target-%[1]s-alerts@example.com"
  %[2]s
}

resource "appwrite_messaging_topic" "test" {
  name = "tf-acc-test-target-%[1]s"
}

resource "appwrite_messaging_subscriber" "test" {
  topic_id  = appwrite_messaging_topic.test.id
  target_id = appwrite_user_target.test.id
}
`, suffix, nameLine)
	}

	acceptance.ResourceTestAllowingReplace(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("Alerts"),
				Check:  resource.TestCheckResourceAttr("appwrite_user_target.test", "name", "Alerts"),
			},
			{
				Config: config(""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("appwrite_user_target.test", plancheck.ResourceActionDestroyBeforeCreate),
						plancheck.ExpectResourceAction("appwrite_messaging_subscriber.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("appwrite_user_target.test", "name"),
					resource.TestCheckResourceAttrPair("appwrite_messaging_subscriber.test", "target_id", "appwrite_user_target.test", "id"),
				),
			},
			acceptance.ExpectEmptyPlanAfterApply(config("")),
		},
	})
}

// TestAccUserTarget_providerOfWrongType covers a provider Appwrite's create
// route would accept without checking. The provider has the server check it,
// and the target it already created is recorded as tainted rather than lost:
// the destroy check confirms Terraform still deletes it.
func TestAccUserTarget_providerOfWrongType(t *testing.T) {
	userID, cleanup := acceptance.User(t)
	defer cleanup()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "appwrite_messaging_provider" "sms" {
  id          = "tf-acc-test-twilio-%[1]s"
  name        = "tf-acc-test twilio"
  type        = "twilio"
  account_sid = "ACtestaccountsid"
  auth_token  = "testauthtoken"
  from        = "+14155550100"
}

resource "appwrite_user_target" "test" {
  user_id       = %[2]q
  provider_type = "email"
  identifier    = "tf-acc-test-%[1]s@example.com"
  provider_id   = appwrite_messaging_provider.sms.id
}
`, suffix, userID),
				ExpectError: regexp.MustCompile(`provider_incorrect_type`),
			},
		},
	})

	// The destroy check only sees what reached state. Had the created target
	// not been recorded, it would be left on the user, so look there too.
	if os.Getenv(resource.EnvTfAcc) == "" {
		return
	}
	targets, err := appwrite.NewUsers(acceptance.ProjectClient(t)).ListTargets(userID)
	if err != nil {
		t.Fatalf("listing the fixture user's targets: %v", err)
	}
	for _, target := range targets.Targets {
		if target.ProviderType != "email" || target.Identifier != fmt.Sprintf("tf-acc-test-%s@example.com", suffix) {
			continue
		}
		t.Errorf("target %s was created and left behind after its provider was refused", target.Id)
	}
}

// TestAccUserTarget_invalidIdentifier checks that an identifier the provider
// type cannot take fails at plan time rather than mid-apply.
func TestAccUserTarget_invalidIdentifier(t *testing.T) {
	config := func(providerType, identifier string) string {
		return fmt.Sprintf(`
resource "appwrite_user_target" "test" {
  user_id       = "tf-acc-test-unused"
  provider_type = %q
  identifier    = %q
}
`, providerType, identifier)
	}

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config("email", "not-an-email"),
				ExpectError: regexp.MustCompile(`email targets take an email address`),
			},
			{
				Config:      config("sms", "4155550100"),
				ExpectError: regexp.MustCompile(`E\.164`),
			},
			{
				Config:      config("fax", "anything"),
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
		},
	})
}

type targetTestConfig struct {
	userID   string
	suffix   string
	email    string
	phone    string
	name     string
	provider string
}

func (c targetTestConfig) String() string {
	return fmt.Sprintf(`
resource "appwrite_messaging_provider" "first" {
  id         = "tf-acc-test-smtp-a-%[1]s"
  name       = "tf-acc-test first"
  type       = "smtp"
  host       = "smtp.example.com"
  port       = 587
  from_email = "noreply@example.com"
}

resource "appwrite_messaging_provider" "second" {
  id         = "tf-acc-test-smtp-b-%[1]s"
  name       = "tf-acc-test second"
  type       = "smtp"
  host       = "smtp.example.com"
  port       = 587
  from_email = "noreply@example.com"
}

resource "appwrite_user_target" "email" {
  user_id       = %[2]q
  provider_type = "email"
  identifier    = %[3]q
  name          = %[5]q
  provider_id   = appwrite_messaging_provider.%[6]s.id
}

resource "appwrite_user_target" "sms" {
  user_id       = %[2]q
  provider_type = "sms"
  identifier    = %[4]q
}

resource "appwrite_user_target" "push" {
  id            = "tf-acc-test-push-%[1]s"
  user_id       = %[2]q
  provider_type = "push"
  identifier    = "tf-acc-test-device-token-%[1]s"
  name          = "Test device"
}

resource "appwrite_messaging_topic" "test" {
  name = "tf-acc-test-target-%[1]s"
}

resource "appwrite_messaging_subscriber" "email" {
  topic_id  = appwrite_messaging_topic.test.id
  target_id = appwrite_user_target.email.id
}

resource "appwrite_messaging_subscriber" "push" {
  topic_id  = appwrite_messaging_topic.test.id
  target_id = appwrite_user_target.push.id
}
`, c.suffix, c.userID, c.email, c.phone, c.name, c.provider)
}

func targetImportID(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}
		return rs.Primary.Attributes["user_id"] + "/" + rs.Primary.Attributes["id"], nil
	}
}
