package user_test

import (
	"fmt"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// Unlike the ephemeral API key, a session is server-side state, so Close can
// genuinely revoke it. This asserts that it does: after the run the session is
// gone, not merely expiring on its own schedule.
func TestAccAuthSessionEphemeral_closedOnFinish(t *testing.T) {
	// The user is created outside Terraform on purpose: if Terraform destroyed
	// it, its sessions would go with it and the check below would pass whether
	// or not Close did anything.
	userID, cleanupUser := acceptance.User(t)
	defer cleanupUser()

	var sessionID string

	resource.Test(t, resource.TestCase{
		PreCheck: func() { acceptance.PreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactoriesWithEcho,
		CheckDestroy: func(*terraform.State) error {
			if sessionID == "" {
				return fmt.Errorf("test did not capture a session")
			}
			usersClient := appwrite.NewUsers(acceptance.ProjectClient(t))
			sessions, err := usersClient.ListSessions(userID)
			if err != nil {
				return fmt.Errorf("listing sessions: %w", err)
			}
			for _, s := range sessions.Sessions {
				if s.Id == sessionID {
					return fmt.Errorf("session %s still exists after close; it should have been deleted", sessionID)
				}
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testAccSessionEphemeralConfig(userID),
				Check: func(st *terraform.State) error {
					sessionID = st.RootModule().Resources["echo.test"].Primary.Attributes["data.id"]
					return nil
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("secret"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("expire"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func testAccSessionEphemeralConfig(userID string) string {
	return fmt.Sprintf(`
ephemeral "appwrite_auth_session" "test" {
  user_id = %q
}

provider "echo" {
  data = ephemeral.appwrite_auth_session.test
}

resource "echo" "test" {}
`, userID)
}
