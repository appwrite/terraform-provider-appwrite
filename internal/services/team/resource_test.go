package team_test

import (
	"encoding/json"
	"fmt"
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

func TestAccTeamResource_basic(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamConfig("engineering", "Engineering"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_auth_team.test", "id", "engineering"),
					resource.TestCheckResourceAttr("appwrite_auth_team.test", "name", "Engineering"),
					resource.TestCheckResourceAttrSet("appwrite_auth_team.test", "created_at"),
					resource.TestCheckResourceAttr("appwrite_auth_team.test", "prefs", "{}"),
				),
			},
			{
				ResourceName:      "appwrite_auth_team.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTeamConfig("engineering", "Engineering Team"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_auth_team.test", "name", "Engineering Team"),
				),
			},
		},
	})
}

func testAccTeamConfig(id, name string) string {
	return fmt.Sprintf(`
resource "appwrite_auth_team" "test" {
  id   = %q
  name = %q
}
`, id, name)
}

// TestAccTeamResource_prefs sets, changes and stops managing preferences, and
// shows that a configuration without prefs never touches the ones on the
// server.
func TestAccTeamResource_prefs(t *testing.T) {
	teamID := "tf-acc-test-" + strings.ToLower(id.Unique())
	const address = "appwrite_auth_team.test"

	config := func(prefs string) string {
		line := ""
		if prefs != "" {
			line = "prefs = " + prefs
		}
		return fmt.Sprintf(`
resource "appwrite_auth_team" "test" {
  id   = %q
  name = "Prefs"
  %s
}
`, teamID, line)
	}
	without := config("")
	// Keys out of order, 1.0 for 1 and an integer beyond 2^53: all spellings
	// Appwrite stores differently, none of which may show as a diff.
	initial := config(`jsonencode({ theme = "dark", seats = 10, ratio = 1.0, big = 9007199254740993, flags = { beta = true }, tags = ["a", "b"] })`)
	changed := config(`jsonencode({ theme = "light" })`)

	setOutOfBand := func(prefs map[string]interface{}) func() {
		return func() {
			if _, err := appwrite.NewTeams(acceptance.ProjectClient(t)).UpdatePrefs(teamID, prefs); err != nil {
				t.Fatalf("setting prefs outside Terraform: %v", err)
			}
		}
	}

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: without,
				Check:  resource.TestCheckResourceAttr(address, "prefs", "{}"),
			},
			{
				// Preferences set outside Terraform are reported, and a
				// configuration without prefs plans nothing about them.
				PreConfig: setOutOfBand(map[string]interface{}{"owner": "console"}),
				Config:    without,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "prefs", `{"owner":"console"}`),
					checkServerPrefs(t, teamID, `{"owner":"console"}`),
				),
			},
			{
				Config: initial,
				Check: resource.ComposeAggregateTestCheckFunc(
					checkServerPrefs(t, teamID, `{"big":9007199254740993,"flags":{"beta":true},"ratio":1,"seats":10,"tags":["a","b"],"theme":"dark"}`),
				),
			},
			acceptance.ExpectEmptyPlanAfterApply(initial),
			{
				ResourceName: address,
				ImportState:  true,
				// Import stores the canonical form, which for jsonencode
				// output is the same string.
				ImportStateVerify: true,
			},
			{
				Config: changed,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate),
					},
				},
				Check: checkServerPrefs(t, teamID, `{"theme":"light"}`),
			},
			{
				// Drift on managed preferences is corrected.
				PreConfig: setOutOfBand(map[string]interface{}{"theme": "drifted"}),
				Config:    changed,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate),
					},
				},
				Check: checkServerPrefs(t, teamID, `{"theme":"light"}`),
			},
			{
				Config:      config(`jsonencode({})`),
				ExpectError: regexp.MustCompile(`Team preferences cannot be cleared`),
			},
			{
				Config:      config(`jsonencode(["not", "an", "object"])`),
				ExpectError: regexp.MustCompile(`prefs must be a JSON object`),
			},
			{
				// Removing prefs stops managing them; it does not clear them.
				// Last, so the post-test destroy has a valid configuration.
				Config: without,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: checkServerPrefs(t, teamID, `{"theme":"light"}`),
			},
		},
	})
}

// TestAccTeamResource_prefsAtCreate sets preferences in the same apply that
// creates the team, and accepts an explicit empty object for a new team.
func TestAccTeamResource_prefsAtCreate(t *testing.T) {
	teamID := "tf-acc-test-" + strings.ToLower(id.Unique())
	emptyID := "tf-acc-test-" + strings.ToLower(id.Unique())
	config := fmt.Sprintf(`
resource "appwrite_auth_team" "test" {
  id    = %q
  name  = "Prefs at create"
  prefs = jsonencode({ plan = "pro", limits = { seats = 5 } })
}

resource "appwrite_auth_team" "empty" {
  id    = %q
  name  = "Empty prefs"
  prefs = jsonencode({})
}

data "appwrite_auth_team" "test" {
  id = appwrite_auth_team.test.id
}
`, teamID, emptyID)

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					checkServerPrefs(t, teamID, `{"limits":{"seats":5},"plan":"pro"}`),
					resource.TestCheckResourceAttr("appwrite_auth_team.empty", "prefs", "{}"),
					resource.TestCheckResourceAttr("data.appwrite_auth_team.test", "prefs", `{"limits":{"seats":5},"plan":"pro"}`),
				),
			},
			acceptance.ExpectEmptyPlanAfterApply(config),
		},
	})
}

// checkServerPrefs asserts what Appwrite holds, independently of state.
func checkServerPrefs(t *testing.T, teamID, want string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		prefs, err := appwrite.NewTeams(acceptance.ProjectClient(t)).GetPrefs(teamID)
		if err != nil {
			return fmt.Errorf("reading prefs of %s: %w", teamID, err)
		}
		var got map[string]json.RawMessage
		if err := prefs.Decode(&got); err != nil {
			return fmt.Errorf("decoding prefs of %s: %w", teamID, err)
		}
		encoded, _ := json.Marshal(got)
		if string(encoded) != want {
			return fmt.Errorf("server prefs of %s = %s, want %s", teamID, encoded, want)
		}
		return nil
	}
}
