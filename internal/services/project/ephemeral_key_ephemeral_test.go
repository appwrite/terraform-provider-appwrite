package project_test

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

func TestAccProjectEphemeralKey_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() { acceptance.PreCheck(t) },
		// Ephemeral resources arrived in Terraform 1.10.
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		// The key never lands in state, so it is echoed into one to be checked.
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactoriesWithEcho,
		Steps: []resource.TestStep{
			{
				Config: `
ephemeral "appwrite_project_ephemeral_key" "test" {
  scopes           = ["users.read"]
  duration_seconds = 900
}

provider "echo" {
  data = ephemeral.appwrite_project_ephemeral_key.test
}

resource "echo" "test" {}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("expire"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("secret"), knownvalue.NotNull()),
					// The scopes come back as asked, which is what bounds the
					// key: it cannot be revoked, so its reach is fixed at mint
					// time by these and by duration_seconds.
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("scopes"),
						knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("users.read")})),
				},
			},
		},
	})
}

// TestAccProjectEphemeralKey_secretStaysOutOfState is the claim the whole
// resource exists to make: the secret reaches the configuration but no state
// file of the resource under test. The echo resource opts in to holding it, so
// it is checked from there rather than asserted to be absent everywhere.
func TestAccProjectEphemeralKey_secretStaysOutOfState(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() { acceptance.PreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactoriesWithEcho,
		Steps: []resource.TestStep{
			{
				Config: `
ephemeral "appwrite_project_ephemeral_key" "test" {
  scopes = ["users.read"]
}

provider "echo" {
  data = ephemeral.appwrite_project_ephemeral_key.test.secret
}

resource "echo" "test" {}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					// An ephemeral key secret is recognisable by its prefix; the
					// provider's own credential detection keys off the same one.
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data"),
						knownvalue.StringRegexp(regexpEphemeralSecret)),
				},
			},
		},
	})
}

var regexpEphemeralSecret = regexp.MustCompile(`^ephemeral_`)
