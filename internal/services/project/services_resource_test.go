package project_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	sdkclient "github.com/appwrite/sdk-for-go/v7/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

// These tests switch services and protocols off in the acceptance project,
// which other tests share. They only touch switches no other test needs for
// requests made without an API key: the JWT and session tests need account and
// REST, so those two are never disabled here.

// TestAccProjectServicesResource covers create, import, update, drift on a
// managed and an unmanaged service, and destroy enabling only what it managed.
func TestAccProjectServicesResource(t *testing.T) {
	// Safety net for an aborted run. It runs after CheckDestroy, so it cannot
	// hide a destroy that failed to enable something.
	t.Cleanup(func() { enableAll(t) })

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			acceptance.CheckDestroy(t),
			// Every service the last configuration managed is enabled again.
			checkServices(t, map[string]bool{"locale": true, "project": true, "advisor": true}),
			// vcs was disabled outside Terraform and never managed, so
			// destroy leaves it alone.
			checkServices(t, map[string]bool{"vcs": false}),
		),
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_project_services" "test" {
  locale  = false
  avatars = false
  project = false
  graphql = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_project_services.test", "id", os.Getenv("APPWRITE_PROJECT_ID")),
					resource.TestCheckResourceAttr("appwrite_project_services.test", "project_id", os.Getenv("APPWRITE_PROJECT_ID")),
					resource.TestCheckResourceAttr("appwrite_project_services.test", "locale", "false"),
					resource.TestCheckResourceAttr("appwrite_project_services.test", "avatars", "false"),
					resource.TestCheckResourceAttr("appwrite_project_services.test", "project", "false"),
					resource.TestCheckResourceAttr("appwrite_project_services.test", "graphql", "true"),
					resource.TestCheckNoResourceAttr("appwrite_project_services.test", "storage"),
					checkServices(t, map[string]bool{"locale": false, "avatars": false, "project": false, "graphql": true, "storage": true}),
					// A disabled service turns away requests without a key and
					// lets keyed ones through. The provider's own requests
					// are keyed, which is why refreshing this resource keeps
					// working with the project service itself disabled.
					checkGuestBlocked(t, "general_service_disabled", func(c sdkclient.Client) error {
						_, err := appwrite.NewLocale(c).Get()
						return err
					}),
					checkKeyAllowed(t, func(c sdkclient.Client) error {
						_, err := appwrite.NewLocale(c).Get()
						return err
					}),
				),
			},
			{
				ResourceName:      "appwrite_project_services.test",
				ImportState:       true,
				ImportStateId:     os.Getenv("APPWRITE_PROJECT_ID"),
				ImportStateVerify: false,
				// Import records every service the server reports, not just
				// the ones the configuration sets.
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("imported %d instances, want 1", len(states))
					}
					want := map[string]string{"locale": "false", "avatars": "false", "project": "false", "graphql": "true", "storage": "true", "account": "true"}
					for key, value := range want {
						if got := states[0].Attributes[key]; got != value {
							return fmt.Errorf("imported %s = %q, want %q", key, got, value)
						}
					}
					return nil
				},
			},
			{
				// locale changes, avatars and graphql leave the configuration
				// and so are enabled, project stays disabled, advisor is new.
				Config: `
resource "appwrite_project_services" "test" {
  locale  = true
  project = false
  advisor = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_project_services.test", "locale", "true"),
					resource.TestCheckNoResourceAttr("appwrite_project_services.test", "avatars"),
					resource.TestCheckNoResourceAttr("appwrite_project_services.test", "graphql"),
					checkServices(t, map[string]bool{"locale": true, "avatars": true, "graphql": true, "project": false, "advisor": false}),
				),
			},
			{
				// A change outside Terraform to a managed service is drift and
				// is corrected. The same change to an unmanaged one is not.
				PreConfig: func() {
					setService(t, "advisor", true)
					setService(t, "vcs", false)
				},
				Config: `
resource "appwrite_project_services" "test" {
  locale  = true
  project = false
  advisor = false
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("appwrite_project_services.test", plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue("appwrite_project_services.test", tfjsonpath.New("advisor"), knownvalue.Bool(false)),
						plancheck.ExpectKnownValue("appwrite_project_services.test", tfjsonpath.New("vcs"), knownvalue.Null()),
					},
				},
				Check: checkServices(t, map[string]bool{"advisor": false, "vcs": false}),
			},
			acceptance.ExpectEmptyPlanAfterApply(`
resource "appwrite_project_services" "test" {
  locale  = true
  project = false
  advisor = false
}
`),
		},
	})
}

func guestClient() sdkclient.Client {
	return appwrite.NewClient(
		appwrite.WithEndpoint(os.Getenv("APPWRITE_ENDPOINT")),
		appwrite.WithProject(os.Getenv("APPWRITE_PROJECT_ID")),
	)
}

func checkGuestBlocked(t *testing.T, errorType string, call func(sdkclient.Client) error) resource.TestCheckFunc {
	t.Helper()
	return func(*terraform.State) error {
		err := call(guestClient())
		if got := common.ErrorType(err); got != errorType {
			return fmt.Errorf("request without a key: got error type %q, want %q: %w", got, errorType, err)
		}
		return nil
	}
}

func checkKeyBlocked(t *testing.T, errorType string, call func(sdkclient.Client) error) resource.TestCheckFunc {
	t.Helper()
	return func(*terraform.State) error {
		err := call(acceptance.ProjectClient(t))
		if got := common.ErrorType(err); got != errorType {
			return fmt.Errorf("request with a key: got error type %q, want %q: %w", got, errorType, err)
		}
		return nil
	}
}

func checkKeyAllowed(t *testing.T, call func(sdkclient.Client) error) resource.TestCheckFunc {
	t.Helper()
	return func(*terraform.State) error {
		if err := call(acceptance.ProjectClient(t)); err != nil {
			return fmt.Errorf("request with a key was refused: %w", err)
		}
		return nil
	}
}

func checkServices(t *testing.T, want map[string]bool) resource.TestCheckFunc {
	t.Helper()
	return func(*terraform.State) error {
		project, err := appwrite.NewProject(acceptance.ProjectClient(t)).Get()
		if err != nil {
			return err
		}
		got := map[string]bool{}
		for _, service := range project.Services {
			got[service.Id] = service.Enabled
		}
		return compareSwitches("service", got, want)
	}
}

func checkProtocols(t *testing.T, want map[string]bool) resource.TestCheckFunc {
	t.Helper()
	return func(*terraform.State) error {
		project, err := appwrite.NewProject(acceptance.ProjectClient(t)).Get()
		if err != nil {
			return err
		}
		got := map[string]bool{}
		for _, protocol := range project.Protocols {
			got[protocol.Id] = protocol.Enabled
		}
		return compareSwitches("protocol", got, want)
	}
}

func compareSwitches(noun string, got, want map[string]bool) error {
	for id, enabled := range want {
		value, ok := got[id]
		if !ok {
			return fmt.Errorf("server does not report %s %q", noun, id)
		}
		if value != enabled {
			return fmt.Errorf("%s %q enabled = %v, want %v", noun, id, value, enabled)
		}
	}
	return nil
}

func setService(t *testing.T, id string, enabled bool) {
	t.Helper()
	if _, err := appwrite.NewProject(acceptance.ProjectClient(t)).UpdateService(id, enabled); err != nil {
		t.Fatalf("setting service %s: %v", id, err)
	}
}

// enableAll turns every service and protocol the server reports back on, so a
// failed or interrupted run does not leave the shared project degraded.
func enableAll(t *testing.T) {
	t.Helper()
	if os.Getenv(resource.EnvTfAcc) == "" {
		return
	}
	client := appwrite.NewProject(acceptance.ProjectClient(t))
	project, err := client.Get()
	if err != nil {
		t.Errorf("reading project to enable its services: %v", err)
		return
	}
	for _, service := range project.Services {
		if !service.Enabled {
			if _, err := client.UpdateService(service.Id, true); err != nil {
				t.Errorf("enabling service %s: %v", service.Id, err)
			}
		}
	}
	for _, protocol := range project.Protocols {
		if !protocol.Enabled {
			if _, err := client.UpdateProtocol(protocol.Id, true); err != nil {
				t.Errorf("enabling protocol %s: %v", protocol.Id, err)
			}
		}
	}
}
