package user_test

import (
	"regexp"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// A JWT is three base64url segments separated by dots. Matching the shape is
// enough to show a real token came back rather than an empty string.
var regexpJWT = regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`)

func TestAccAuthJWTEphemeral_basic(t *testing.T) {
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
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data"),
						knownvalue.StringRegexp(regexpJWT)),
				},
			},
		},
	})
}

func testAccJWTEphemeralConfig(userID string) string {
	return `
ephemeral "appwrite_auth_jwt" "test" {
  user_id          = "` + userID + `"
  duration_seconds = 900
}

provider "echo" {
  data = ephemeral.appwrite_auth_jwt.test.jwt
}

resource "echo" "test" {}
`
}
