package project_test

import (
	"fmt"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/id"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

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
				},
			},
		},
	})
}

// TestAccProjectEphemeralKey_secretIsUsableWithinItsScopes checks the two
// things a caller depends on: the minted key actually authenticates, and it
// cannot reach past the scopes it was asked for. Matching the token's format
// instead would pass for a key that authenticates nowhere.
func TestAccProjectEphemeralKey_secretIsUsableWithinItsScopes(t *testing.T) {
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
				Check: func(st *terraform.State) error {
					secret := st.RootModule().Resources["echo.test"].Primary.Attributes["data"]
					if secret == "" {
						return fmt.Errorf("no secret was echoed")
					}

					clt := acceptance.ClientWithKey(t, secret)

					// users.read was granted, so this must work.
					if _, err := appwrite.NewUsers(clt).List(); err != nil {
						return fmt.Errorf("granted scope users.read was refused: %w", err)
					}

					// buckets.write was not, so this must not.
					if _, err := appwrite.NewStorage(clt).CreateBucket(id.Unique(), "should-not-exist"); err == nil {
						return fmt.Errorf("key created a bucket despite not being granted buckets.write")
					}
					return nil
				},
			},
		},
	})
}
