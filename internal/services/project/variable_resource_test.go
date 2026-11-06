package project_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

const projectVariableName = "appwrite_project_variable.test"

// Keys are unique within a project and every run shares one, so each test
// takes a fresh key. The TF_ACC_TEST_ prefix is what the sweeper matches.
func testAccProjectVariableKey(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("TF_ACC_TEST_%d", time.Now().UnixNano())
}

func testAccProjectVariableConfig(key, value, secret string) string {
	secretLine := ""
	if secret != "" {
		secretLine = "\n  secret = " + secret
	}
	return fmt.Sprintf(`
resource "appwrite_project_variable" "test" {
  key   = %q
  value = %q%s
}
`, key, value, secretLine)
}

// testAccCheckProjectVariableOnServer asserts what the server holds, not what
// state says it holds: a non-secret value comes back from the API, and a
// secret one comes back empty.
func testAccCheckProjectVariableOnServer(t *testing.T, key, value string, secret bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[projectVariableName]
		if !ok {
			return fmt.Errorf("%s not in state", projectVariableName)
		}
		variable, err := appwrite.NewProject(acceptance.ProjectClient(t)).GetVariable(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("reading variable %s: %w", rs.Primary.ID, err)
		}
		if variable.Key != key || variable.Secret != secret {
			return fmt.Errorf("server has key %q secret %v, want key %q secret %v", variable.Key, variable.Secret, key, secret)
		}
		want := value
		if secret {
			want = ""
		}
		if variable.Value != want {
			return fmt.Errorf("server returned value %q, want %q", variable.Value, want)
		}
		return nil
	}
}

func TestAccProjectVariableResource_basic(t *testing.T) {
	key := testAccProjectVariableKey(t)
	renamed := key + "_RENAMED"

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectVariableConfig(key, "https://api.example.com", "false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(projectVariableName, "id"),
					resource.TestCheckResourceAttr(projectVariableName, "key", key),
					resource.TestCheckResourceAttr(projectVariableName, "value", "https://api.example.com"),
					resource.TestCheckResourceAttr(projectVariableName, "secret", "false"),
					resource.TestCheckResourceAttrSet(projectVariableName, "created_at"),
					resource.TestCheckResourceAttrSet(projectVariableName, "project_id"),
					testAccCheckProjectVariableOnServer(t, key, "https://api.example.com", false),
				),
			},
			{
				// The value is imported only through the configuration, so it
				// can come from value_wo instead.
				ResourceName:            projectVariableName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"value"},
			},
			{
				ResourceName: projectVariableName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources[projectVariableName]
					return rs.Primary.Attributes["project_id"] + "/" + rs.Primary.ID, nil
				},
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"value"},
			},
			{
				Config: testAccProjectVariableConfig(key, "https://api.example.com/v2", "false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(projectVariableName, "value", "https://api.example.com/v2"),
					testAccCheckProjectVariableOnServer(t, key, "https://api.example.com/v2", false),
				),
			},
			{
				Config: testAccProjectVariableConfig(renamed, "https://api.example.com/v2", "false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(projectVariableName, "key", renamed),
					testAccCheckProjectVariableOnServer(t, renamed, "https://api.example.com/v2", false),
				),
			},
			{
				// Making a variable secret is allowed in place; ResourceTest
				// fails the step if it plans a replacement.
				Config: testAccProjectVariableConfig(renamed, "hidden", "true"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(projectVariableName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(projectVariableName, "secret", "true"),
					resource.TestCheckResourceAttr(projectVariableName, "value", "hidden"),
					testAccCheckProjectVariableOnServer(t, renamed, "", true),
				),
			},
			{
				// Updating a secret's value works although it cannot be read.
				Config: testAccProjectVariableConfig(renamed, "hidden-v2", "true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(projectVariableName, "value", "hidden-v2"),
				),
			},
			acceptance.ExpectEmptyPlanAfterApply(testAccProjectVariableConfig(renamed, "hidden-v2", "true")),
		},
	})
}

// Appwrite makes a variable secret unless told otherwise, and rejects making a
// secret one non-secret. Omitting secret therefore reads back as true, and
// setting it to false afterwards has to recreate the variable rather than plan
// an update that can only fail.
func TestAccProjectVariableResource_unsetSecretReplaces(t *testing.T) {
	key := testAccProjectVariableKey(t)
	var firstID string

	acceptance.ResourceTestAllowingReplace(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectVariableConfig(key, "default", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(projectVariableName, "secret", "true"),
					resource.TestCheckResourceAttr(projectVariableName, "value", "default"),
					testAccCheckProjectVariableOnServer(t, key, "", true),
					func(s *terraform.State) error {
						firstID = s.RootModule().Resources[projectVariableName].Primary.ID
						return nil
					},
				),
			},
			acceptance.ExpectEmptyPlanAfterApply(testAccProjectVariableConfig(key, "default", "")),
			{
				Config: testAccProjectVariableConfig(key, "default", "false"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(projectVariableName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(projectVariableName, "secret", "false"),
					testAccCheckProjectVariableOnServer(t, key, "default", false),
					func(s *terraform.State) error {
						if id := s.RootModule().Resources[projectVariableName].Primary.ID; id == firstID {
							return fmt.Errorf("variable kept ID %s; want it recreated", id)
						}
						return nil
					},
				),
			},
		},
	})
}

// A non-secret variable has its value returned by the API on both create and
// read, so this is the case where a value_wo value could leak into state.
func TestAccProjectVariableResource_writeOnlyValueStaysOutOfState(t *testing.T) {
	key := testAccProjectVariableKey(t)
	config := func(value string, version int) string {
		return fmt.Sprintf(`
resource "appwrite_project_variable" "test" {
  key              = %q
  secret           = false
  value_wo         = %q
  value_wo_version = %d
}
`, key, value, version)
	}

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck: func() { acceptance.PreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("wo-value-one", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(projectVariableName, "value"),
					resource.TestCheckNoResourceAttr(projectVariableName, "value_wo"),
					resource.TestCheckResourceAttr(projectVariableName, "value_wo_version", "1"),
					testAccCheckProjectVariableOnServer(t, key, "wo-value-one", false),
				),
			},
			{
				Config: config("wo-value-two", 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(projectVariableName, "value"),
					resource.TestCheckResourceAttr(projectVariableName, "value_wo_version", "2"),
					testAccCheckProjectVariableOnServer(t, key, "wo-value-two", false),
				),
			},
		},
	})
}
