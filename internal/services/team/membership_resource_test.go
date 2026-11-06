package team_test

import (
	"fmt"
	"math/rand/v2"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/id"
	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const membershipAddress = "appwrite_auth_team_membership.test"

func membershipImportID(s *terraform.State) (string, error) {
	rs, ok := s.RootModule().Resources[membershipAddress]
	if !ok {
		return "", fmt.Errorf("resource %s not found in state", membershipAddress)
	}
	return rs.Primary.Attributes["team_id"] + "/" + rs.Primary.Attributes["id"], nil
}

func TestAccTeamMembership_userID(t *testing.T) {
	suffix := strings.ToLower(id.Unique())
	userID := "tf-acc-test-" + suffix
	config := func(roles string) string {
		return fmt.Sprintf(`
resource "appwrite_auth_team" "test" {
  name = "tf-acc-test-%[1]s"
}

resource "appwrite_auth_user" "test" {
  id    = %[2]q
  name  = "Member %[1]s"
  email = "%[2]s@example.com"
}

resource "appwrite_auth_team_membership" "test" {
  team_id = appwrite_auth_team.test.id
  user_id = appwrite_auth_user.test.id
  roles   = %[3]s
}
`, suffix, userID, roles)
	}

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(`["developer"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(membershipAddress, "id"),
					resource.TestCheckResourceAttr(membershipAddress, "user_id", userID),
					resource.TestCheckResourceAttr(membershipAddress, "roles.#", "1"),
					resource.TestCheckTypeSetElemAttr(membershipAddress, "roles.*", "developer"),
					// Created with an API key, a membership is confirmed at once.
					resource.TestCheckResourceAttr(membershipAddress, "confirm", "true"),
					resource.TestCheckResourceAttrSet(membershipAddress, "joined_at"),
					resource.TestCheckResourceAttr(membershipAddress, "user_name", "Member "+suffix),
					resource.TestCheckResourceAttr(membershipAddress, "user_email", userID+"@example.com"),
					resource.TestCheckResourceAttr(membershipAddress, "user_phone", ""),
					resource.TestCheckNoResourceAttr(membershipAddress, "email"),
					resource.TestCheckResourceAttrSet(membershipAddress, "created_at"),
				),
			},
			{
				ResourceName:      membershipAddress,
				ImportState:       true,
				ImportStateIdFunc: membershipImportID,
				ImportStateVerify: true,
			},
			{
				// Roles change in place; the harness fails the step on a
				// replacement.
				Config: config(`["lead", "developer"]`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(membershipAddress, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(membershipAddress, "roles.#", "2"),
					resource.TestCheckTypeSetElemAttr(membershipAddress, "roles.*", "lead"),
					resource.TestCheckTypeSetElemAttr(membershipAddress, "roles.*", "developer"),
				),
			},
			{
				// Order does not matter.
				Config: config(`["developer", "lead"]`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: config(`[]`),
				Check:  resource.TestCheckResourceAttr(membershipAddress, "roles.#", "0"),
			},
			acceptance.ExpectEmptyPlanAfterApply(config(`[]`)),
		},
	})
}

// TestAccTeamMembership_email covers a member given by an address no user
// has. Appwrite creates the user, and destroying the membership leaves it
// behind; the cleanup asserts both.
func TestAccTeamMembership_email(t *testing.T) {
	suffix := strings.ToLower(id.Unique())
	// Mixed case: Appwrite matches and stores the address in lowercase, while
	// the configured spelling stays in state without a diff.
	email := fmt.Sprintf("TF-Acc-Test-%s@Example.com", suffix)
	config := func(roles string) string {
		return fmt.Sprintf(`
resource "appwrite_auth_team" "test" {
  name = "tf-acc-test-%[1]s"
}

resource "appwrite_auth_team_membership" "test" {
  team_id = appwrite_auth_team.test.id
  email   = %[2]q
  name    = "tf-acc-test invitee %[1]s"
  roles   = %[3]s
}
`, suffix, email, roles)
	}

	createdUser := cleanupCreatedUser(t, true)

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(`["viewer"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(membershipAddress, "email", email),
					resource.TestCheckResourceAttr(membershipAddress, "user_email", strings.ToLower(email)),
					resource.TestCheckResourceAttr(membershipAddress, "user_name", "tf-acc-test invitee "+suffix),
					resource.TestCheckResourceAttr(membershipAddress, "confirm", "true"),
					resource.TestCheckResourceAttrSet(membershipAddress, "user_id"),
					createdUser.capture,
				),
			},
			{
				ResourceName:      membershipAddress,
				ImportState:       true,
				ImportStateIdFunc: membershipImportID,
				ImportStateVerify: true,
				// What the membership was created from cannot be recovered.
				ImportStateVerifyIgnore: []string{"email", "name"},
			},
			{
				Config: config(`["viewer", "editor"]`),
				Check:  resource.TestCheckResourceAttr(membershipAddress, "roles.#", "2"),
			},
			acceptance.ExpectEmptyPlanAfterApply(config(`["viewer", "editor"]`)),
		},
	})
}

func TestAccTeamMembership_phone(t *testing.T) {
	suffix := strings.ToLower(id.Unique())
	// Random so reruns do not collide; the server only checks the shape.
	phone := fmt.Sprintf("+1555%07d", rand.IntN(10_000_000))
	config := fmt.Sprintf(`
resource "appwrite_auth_team" "test" {
  name = "tf-acc-test-%[1]s"
}

resource "appwrite_auth_team_membership" "test" {
  team_id = appwrite_auth_team.test.id
  phone   = %[2]q
  # Gives the created user a name the sweeper recognizes.
  name    = "tf-acc-test phone %[1]s"
  roles   = ["support"]
}
`, suffix, phone)

	createdUser := cleanupCreatedUser(t, true)

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(membershipAddress, "phone", phone),
					// Read back from the get endpoint; the create response
					// leaves it empty.
					resource.TestCheckResourceAttr(membershipAddress, "user_phone", phone),
					resource.TestCheckResourceAttr(membershipAddress, "user_email", ""),
					resource.TestCheckResourceAttr(membershipAddress, "confirm", "true"),
					createdUser.capture,
				),
			},
			acceptance.ExpectEmptyPlanAfterApply(config),
		},
	})
}

// TestAccTeamMembership_import adopts a membership created outside Terraform.
// The configuration names the member by email, which import cannot recover,
// so the first plan sets it in place rather than replacing the membership --
// after checking it is the right member.
func TestAccTeamMembership_import(t *testing.T) {
	teamID, userID, email, membershipID := membershipFixture(t)

	config := func(by string) string {
		return fmt.Sprintf(`
resource "appwrite_auth_team_membership" "test" {
  team_id = %[1]q
  %[2]s
  roles   = ["owner"]
}
`, teamID, by)
	}
	byEmail := config(fmt.Sprintf("email = %q", email))

	// The last step asserts a replacement, so the harness's no-replace guard
	// is off; the adoption step asserts its in-place update explicitly.
	acceptance.ResourceTestAllowingReplace(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Creating a membership the user already has fails, and says
				// what to import.
				Config: config(fmt.Sprintf("user_id = %q", userID)),
				// Terraform wraps the message, so any whitespace may be a
				// line break.
				ExpectError: regexp.MustCompile(`import\s+it\s+with\s+the\s+ID\s+` +
					regexp.QuoteMeta(teamID+"/"+membershipID)),
			},
			{
				Config:             byEmail,
				ResourceName:       membershipAddress,
				ImportState:        true,
				ImportStateId:      teamID + "/" + membershipID,
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 imported instance, got %d", len(states))
					}
					attrs := states[0].Attributes
					if attrs["user_id"] != userID || attrs["roles.#"] != "1" || attrs["confirm"] != "true" {
						return fmt.Errorf("unexpected imported attributes: %v", attrs)
					}
					return nil
				},
			},
			{
				// An address belonging to someone else is refused rather
				// than adopted.
				Config:      config(`email = "someone-else@example.com"`),
				ExpectError: regexp.MustCompile(`does not match the imported membership`),
			},
			{
				Config: byEmail,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(membershipAddress, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(membershipAddress, "id", membershipID),
					resource.TestCheckResourceAttr(membershipAddress, "email", email),
				),
			},
			acceptance.ExpectEmptyPlanAfterApply(byEmail),
			{
				// Adopted, email now behaves like any creation argument.
				Config: config(`email = "tf-acc-test-other@example.com"`),
				// A plan-only step's plan is checked here, not in PreApply.
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(membershipAddress, plancheck.ResourceActionReplace),
					},
				},
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccTeamMembership_invalid(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_auth_team_membership" "test" {
  team_id = "team"
  roles   = []
}
`,
				ExpectError: regexp.MustCompile(`(?s)Exactly one of these attributes must be configured`),
			},
			{
				Config: `
resource "appwrite_auth_team_membership" "test" {
  team_id = "team"
  user_id = "user"
  email   = "user@example.com"
  roles   = []
}
`,
				ExpectError: regexp.MustCompile(`(?s)Exactly one of these attributes must be configured`),
			},
			{
				Config: `
resource "appwrite_auth_team_membership" "test" {
  team_id = "team"
  user_id = "user"
  roles   = ["has space"]
}
`,
				ExpectError: regexp.MustCompile(`must be 1 to 81 letters`),
			},
		},
	})
}

type createdUserCleanup struct {
	userID string
}

// capture records the user Appwrite created for the membership, for the
// cleanup to find.
func (c *createdUserCleanup) capture(s *terraform.State) error {
	rs, ok := s.RootModule().Resources[membershipAddress]
	if !ok {
		return fmt.Errorf("resource %s not found in state", membershipAddress)
	}
	c.userID = rs.Primary.Attributes["user_id"]
	return nil
}

// cleanupCreatedUser deletes, once the test is over, the user Appwrite created
// for a membership given by email or phone. With mustSurvive it first asserts
// the user outlived the membership's destroy, which is documented behavior.
func cleanupCreatedUser(t *testing.T, mustSurvive bool) *createdUserCleanup {
	t.Helper()
	c := &createdUserCleanup{}
	t.Cleanup(func() {
		if c.userID == "" {
			return
		}
		users := appwrite.NewUsers(acceptance.ProjectClient(t))
		if mustSurvive && !t.Failed() {
			if _, err := users.Get(c.userID); err != nil {
				t.Errorf("user %s created for the membership did not survive its destroy: %v", c.userID, err)
			}
		}
		if _, err := users.Delete(c.userID); err != nil {
			t.Logf("could not delete user %s created for the membership: %v", c.userID, err)
		}
	})
	return c
}

// membershipFixture creates a team, a user and a membership joining them
// through the client, standing in for a membership that predates Terraform.
func membershipFixture(t *testing.T) (teamID, userID, email, membershipID string) {
	t.Helper()
	// Fixtures run before resource.Test, outside the TF_ACC gate it applies.
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("acceptance tests skipped unless %s is set", resource.EnvTfAcc)
	}
	acceptance.PreCheck(t)

	client := acceptance.ProjectClient(t)
	teamsClient := appwrite.NewTeams(client)
	users := appwrite.NewUsers(client)

	suffix := strings.ToLower(id.Unique())
	teamID = "tf-acc-test-" + suffix
	userID = "tf-acc-test-" + suffix
	email = userID + "@example.com"

	if _, err := teamsClient.Create(teamID, teamID); err != nil {
		t.Fatalf("creating fixture team: %v", err)
	}
	t.Cleanup(func() {
		if _, err := teamsClient.Delete(teamID); err != nil {
			t.Logf("could not delete fixture team %s: %v", teamID, err)
		}
	})
	if _, err := users.Create(userID, users.WithCreateEmail(email)); err != nil {
		t.Fatalf("creating fixture user: %v", err)
	}
	t.Cleanup(func() {
		if _, err := users.Delete(userID); err != nil {
			t.Logf("could not delete fixture user %s: %v", userID, err)
		}
	})
	membership, err := teamsClient.CreateMembership(teamID, []string{"owner"}, teamsClient.WithCreateMembershipUserId(userID))
	if err != nil {
		t.Fatalf("creating fixture membership: %v", err)
	}
	return teamID, userID, email, membership.Id
}
