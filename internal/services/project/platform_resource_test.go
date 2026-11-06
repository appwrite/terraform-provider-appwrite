package project_test

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
)

// TestAccProjectPlatform_allTypes creates one platform of every type, changes
// every name and identifier in place, and imports each one back.
func TestAccProjectPlatform_allTypes(t *testing.T) {
	// Platform IDs are unique across the whole Appwrite instance, so a fixed
	// custom ID would collide with a concurrent or interrupted run.
	webID := "tf-acc-test-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	projectID := os.Getenv("APPWRITE_PROJECT_ID")

	identifiers := map[string]string{
		"web":     "hostname",
		"android": "application_id",
		"apple":   "bundle_identifier",
		"linux":   "package_name",
		"windows": "package_identifier_name",
	}
	// checkOnly asserts that each platform holds its own type's identifier
	// and none of the others.
	checkOnly := func() resource.TestCheckFunc {
		var checks []resource.TestCheckFunc
		for platformType := range identifiers {
			for otherType, attribute := range identifiers {
				if otherType != platformType {
					checks = append(checks, resource.TestCheckNoResourceAttr("appwrite_project_platform."+platformType, attribute))
				}
			}
		}
		return resource.ComposeAggregateTestCheckFunc(checks...)
	}

	steps := make([]resource.TestStep, 0, 3+2*len(identifiers))
	steps = append(steps,
		resource.TestStep{
			Config: testAccProjectPlatformsConfig(webID, "v1", "app.example.com", "com.example.app"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("appwrite_project_platform.web", "id", webID),
				resource.TestCheckResourceAttr("appwrite_project_platform.web", "type", "web"),
				resource.TestCheckResourceAttr("appwrite_project_platform.web", "name", "tf-acc-test web v1"),
				resource.TestCheckResourceAttr("appwrite_project_platform.web", "hostname", "app.example.com"),
				resource.TestCheckResourceAttr("appwrite_project_platform.web", "project_id", projectID),
				resource.TestCheckResourceAttrSet("appwrite_project_platform.web", "created_at"),
				resource.TestCheckResourceAttrSet("appwrite_project_platform.web", "updated_at"),

				resource.TestCheckResourceAttrSet("appwrite_project_platform.android", "id"),
				resource.TestCheckResourceAttr("appwrite_project_platform.android", "type", "android"),
				resource.TestCheckResourceAttr("appwrite_project_platform.android", "application_id", "com.example.app"),
				resource.TestCheckResourceAttr("appwrite_project_platform.apple", "type", "apple"),
				resource.TestCheckResourceAttr("appwrite_project_platform.apple", "bundle_identifier", "com.example.app"),
				resource.TestCheckResourceAttr("appwrite_project_platform.linux", "type", "linux"),
				resource.TestCheckResourceAttr("appwrite_project_platform.linux", "package_name", "com.example.app"),
				resource.TestCheckResourceAttr("appwrite_project_platform.windows", "type", "windows"),
				resource.TestCheckResourceAttr("appwrite_project_platform.windows", "package_identifier_name", "com.example.app"),
				checkOnly(),
			),
		},
		resource.TestStep{
			// Every name and identifier changes; ResourceTest fails the step
			// if any of it plans a replacement. The hostname is stored as
			// written, wildcard and case included.
			Config: testAccProjectPlatformsConfig(webID, "v2", "*.Example.COM", "com.example.renamed"),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("appwrite_project_platform.web", plancheck.ResourceActionUpdate),
					plancheck.ExpectResourceAction("appwrite_project_platform.android", plancheck.ResourceActionUpdate),
					plancheck.ExpectResourceAction("appwrite_project_platform.apple", plancheck.ResourceActionUpdate),
					plancheck.ExpectResourceAction("appwrite_project_platform.linux", plancheck.ResourceActionUpdate),
					plancheck.ExpectResourceAction("appwrite_project_platform.windows", plancheck.ResourceActionUpdate),
				},
			},
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("appwrite_project_platform.web", "id", webID),
				resource.TestCheckResourceAttr("appwrite_project_platform.web", "name", "tf-acc-test web v2"),
				resource.TestCheckResourceAttr("appwrite_project_platform.web", "hostname", "*.Example.COM"),
				resource.TestCheckResourceAttr("appwrite_project_platform.android", "name", "tf-acc-test android v2"),
				resource.TestCheckResourceAttr("appwrite_project_platform.android", "application_id", "com.example.renamed"),
				resource.TestCheckResourceAttr("appwrite_project_platform.apple", "bundle_identifier", "com.example.renamed"),
				resource.TestCheckResourceAttr("appwrite_project_platform.linux", "package_name", "com.example.renamed"),
				resource.TestCheckResourceAttr("appwrite_project_platform.windows", "package_identifier_name", "com.example.renamed"),
				checkOnly(),
			),
		},
		acceptance.ExpectEmptyPlanAfterApply(testAccProjectPlatformsConfig(webID, "v2", "*.Example.COM", "com.example.renamed")),
	)
	for platformType := range identifiers {
		address := "appwrite_project_platform." + platformType
		steps = append(steps,
			resource.TestStep{
				ResourceName:      address,
				ImportState:       true,
				ImportStateIdFunc: platformImportID(address, projectID),
				ImportStateVerify: true,
			},
			resource.TestStep{
				// The short form takes the project from the provider.
				ResourceName:      address,
				ImportState:       true,
				ImportStateVerify: true,
			},
		)
	}

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps:                    steps,
	})
}

// TestAccProjectPlatform_changeType checks that the one change no update
// route can make, the platform's type, replaces it.
func TestAccProjectPlatform_changeType(t *testing.T) {
	acceptance.ResourceTestAllowingReplace(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_project_platform" "test" {
  type     = "apple"
  name     = "tf-acc-test apple"
  bundle_identifier = "com.example.app"
}
`,
				Check: resource.TestCheckResourceAttr("appwrite_project_platform.test", "bundle_identifier", "com.example.app"),
			},
			{
				Config: `
resource "appwrite_project_platform" "test" {
  type           = "android"
  name           = "tf-acc-test apple"
  application_id = "com.example.app"
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("appwrite_project_platform.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_project_platform.test", "type", "android"),
					resource.TestCheckResourceAttr("appwrite_project_platform.test", "application_id", "com.example.app"),
					resource.TestCheckNoResourceAttr("appwrite_project_platform.test", "bundle_identifier"),
				),
			},
		},
	})
}

// TestAccProjectPlatform_invalidConfig checks that what the API would reject
// mid-apply is reported at plan time instead.
func TestAccProjectPlatform_invalidConfig(t *testing.T) {
	config := func(body string) string {
		return fmt.Sprintf("resource \"appwrite_project_platform\" \"test\" {\n%s\n}\n", body)
	}
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config(`type = "web"` + "\n" + `name = "tf-acc-test"`),
				ExpectError: regexp.MustCompile(`web platforms require hostname`),
			},
			{
				Config: config(`type = "android"` + "\n" + `name = "tf-acc-test"` + "\n" +
					`application_id = "com.example.app"` + "\n" + `hostname = "app.example.com"`),
				ExpectError: regexp.MustCompile(`hostname is used only by web platforms`),
			},
			{
				Config:      config(`type = "web"` + "\n" + `name = "tf-acc-test"` + "\n" + `hostname = "https://app.example.com"`),
				ExpectError: regexp.MustCompile(`without a scheme, port or path`),
			},
			{
				Config:      config(`type = "linux"` + "\n" + `name = "   "` + "\n" + `package_name = "example"`),
				ExpectError: regexp.MustCompile(`must not be empty or blank`),
			},
			{
				Config:      config(`type = "ios"` + "\n" + `name = "tf-acc-test"`),
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
		},
	})
}

func platformImportID(address, projectID string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		rs, ok := state.RootModule().Resources[address]
		if !ok || rs.Primary == nil {
			return "", fmt.Errorf("%s not found in state", address)
		}
		return projectID + "/" + rs.Primary.ID, nil
	}
}

// testAccProjectPlatformsConfig declares one platform of each type. The web
// platform takes a custom ID; the others are generated.
func testAccProjectPlatformsConfig(webID, version, hostname, identifier string) string {
	return fmt.Sprintf(`
resource "appwrite_project_platform" "web" {
  id       = %[1]q
  type     = "web"
  name     = "tf-acc-test web %[2]s"
  hostname = %[3]q
}

resource "appwrite_project_platform" "android" {
  type           = "android"
  name           = "tf-acc-test android %[2]s"
  application_id = %[4]q
}

resource "appwrite_project_platform" "apple" {
  type              = "apple"
  name              = "tf-acc-test apple %[2]s"
  bundle_identifier = %[4]q
}

resource "appwrite_project_platform" "linux" {
  type         = "linux"
  name         = "tf-acc-test linux %[2]s"
  package_name = %[4]q
}

resource "appwrite_project_platform" "windows" {
  type                    = "windows"
  name                    = "tf-acc-test windows %[2]s"
  package_identifier_name = %[4]q
}
`, webID, version, hostname, identifier)
}
