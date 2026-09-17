package user_test

import (
	"fmt"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// What a caller depends on is that the token authenticates, and as the right
// user. Matching its three-segment encoding would pass for a token that
// authenticates as nobody.
func TestAccAuthJWTEphemeral_authenticatesAsTheUser(t *testing.T) {
	userID, cleanup := acceptance.User(t)
	defer cleanup()

	resource.Test(t, resource.TestCase{
		PreCheck: func() { acceptance.PreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactoriesWithEcho,
		Steps: []resource.TestStep{
			{
				Config: testAccJWTEphemeralConfig(userID),
				Check: func(st *terraform.State) error {
					jwt := st.RootModule().Resources["echo.test"].Primary.Attributes["data"]
					if jwt == "" {
						return fmt.Errorf("no JWT was echoed")
					}

					account, err := appwrite.NewAccount(acceptance.ClientWithJWT(t, jwt)).Get()
					if err != nil {
						return fmt.Errorf("JWT did not authenticate: %w", err)
					}
					if account.Id != userID {
						return fmt.Errorf("JWT authenticated as %s, want %s", account.Id, userID)
					}
					return nil
				},
			},
		},
	})
}

func testAccJWTEphemeralConfig(userID string) string {
	return fmt.Sprintf(`
ephemeral "appwrite_auth_jwt" "test" {
  user_id          = %q
  duration_seconds = 900
}

provider "echo" {
  data = ephemeral.appwrite_auth_jwt.test.jwt
}

resource "echo" "test" {}
`, userID)
}
