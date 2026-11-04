package organization_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
)

// membershipPreCheck gates the membership tests. Organization memberships are
// served only by Appwrite Cloud, and adding one needs a real account the test
// cannot make up: an email address no account uses would make Appwrite create
// an account, which removing the membership does not delete. The account is
// named by APPWRITE_TEST_MEMBER_EMAIL, and must not already be a member of the
// organization. The organization key must have been created by an owner, or
// it cannot hold organization.memberships.write.
func membershipPreCheck(t *testing.T) {
	t.Helper()
	acceptance.CloudPreCheck(t)
	acceptance.OrganizationPreCheck(t)
	if os.Getenv("APPWRITE_TEST_MEMBER_EMAIL") == "" {
		t.Skip("APPWRITE_TEST_MEMBER_EMAIL must name an existing Appwrite Cloud account, not a member of the organization, to run organization membership acceptance tests")
	}
}

func TestAccOrganizationMembership_email(t *testing.T) {
	email := os.Getenv("APPWRITE_TEST_MEMBER_EMAIL")
	organizationID := os.Getenv("APPWRITE_ORGANIZATION_ID")
	// Written in another case than Appwrite stores, which must not plan a
	// replacement after import.
	configured := capitalized(email)

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { membershipPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMembershipEmailConfig(configured, `["developer"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_organization_membership.test", "organization_id", organizationID),
					resource.TestCheckResourceAttr("appwrite_organization_membership.test", "email", configured),
					resource.TestCheckResourceAttr("appwrite_organization_membership.test", "user_email", strings.ToLower(email)),
					resource.TestCheckResourceAttrSet("appwrite_organization_membership.test", "user_id"),
					resource.TestCheckResourceAttr("appwrite_organization_membership.test", "roles.#", "1"),
					resource.TestCheckTypeSetElemAttr("appwrite_organization_membership.test", "roles.*", "developer"),
					// Added with a key, the member joins without an invitation.
					resource.TestCheckResourceAttr("appwrite_organization_membership.test", "confirm", "true"),
					resource.TestCheckResourceAttrSet("appwrite_organization_membership.test", "joined_at"),
				),
			},
			{
				ResourceName:      "appwrite_organization_membership.test",
				ImportState:       true,
				ImportStateIdFunc: importID("appwrite_organization_membership.test", organizationID),
				ImportStateVerify: true,
				// Imported from the stored, lowercase address.
				ImportStateVerifyIgnore: []string{"email"},
			},
			{
				// Roles change in place.
				Config: testAccMembershipEmailConfig(configured, `["analyst", "billing"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_organization_membership.test", "roles.#", "2"),
					resource.TestCheckTypeSetElemAttr("appwrite_organization_membership.test", "roles.*", "analyst"),
					resource.TestCheckTypeSetElemAttr("appwrite_organization_membership.test", "roles.*", "billing"),
				),
			},
			acceptance.ExpectEmptyPlanAfterApply(testAccMembershipEmailConfig(configured, `["analyst", "billing"]`)),
		},
	})
}

// Adding the same account by user ID, read from a first apply. The second
// step swaps email for user_id in the configuration, which must not replace
// the membership: both name the same account.
func TestAccOrganizationMembership_userID(t *testing.T) {
	email := os.Getenv("APPWRITE_TEST_MEMBER_EMAIL")
	userID := os.Getenv("APPWRITE_TEST_MEMBER_USER_ID")

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck: func() {
			membershipPreCheck(t)
			if userID == "" {
				t.Skip("APPWRITE_TEST_MEMBER_USER_ID must be the user ID of APPWRITE_TEST_MEMBER_EMAIL's account to run this test")
			}
		},
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMembershipUserIDConfig(userID, `["editor"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_organization_membership.test", "user_id", userID),
					resource.TestCheckResourceAttr("appwrite_organization_membership.test", "email", strings.ToLower(email)),
					resource.TestCheckResourceAttr("appwrite_organization_membership.test", "confirm", "true"),
				),
			},
			acceptance.ExpectEmptyPlanAfterApply(testAccMembershipEmailConfig(strings.ToLower(email), `["editor"]`)),
		},
	})
}

func testAccMembershipEmailConfig(email, roles string) string {
	return fmt.Sprintf(`
resource "appwrite_organization_membership" "test" {
  email = %q
  roles = %s
}
`, email, roles)
}

func testAccMembershipUserIDConfig(userID, roles string) string {
	return fmt.Sprintf(`
resource "appwrite_organization_membership" "test" {
  user_id = %q
  roles   = %s
}
`, userID, roles)
}

// capitalized upper-cases the first letter. Evaluated before the precheck
// runs, so it must cope with an unset address.
func capitalized(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// importID builds an import ID from the resource's id, prefixed with the
// organization ID.
func importID(resourceName, organizationID string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("%s not found in state", resourceName)
		}
		return organizationID + "/" + rs.Primary.ID, nil
	}
}
