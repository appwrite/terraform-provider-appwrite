package project_test

import (
	"os"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	sdkclient "github.com/appwrite/sdk-for-go/v7/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
)

// TestAccProjectProtocolsResource covers the protocols, including that
// disabling GraphQL, unlike REST, also turns away requests made with a key.
func TestAccProjectProtocolsResource(t *testing.T) {
	t.Cleanup(func() { enableAll(t) })

	graphql := func(c sdkclient.Client) error {
		_, err := appwrite.NewGraphql(c).Query(map[string]any{"query": "query { localeListCodes { total } }"})
		return err
	}

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			acceptance.CheckDestroy(t),
			checkProtocols(t, map[string]bool{"rest": true, "graphql": true, "websocket": true}),
			checkKeyAllowed(t, graphql),
		),
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_project_protocols" "test" {
  rest      = true
  graphql   = false
  websocket = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_project_protocols.test", "rest", "true"),
					resource.TestCheckResourceAttr("appwrite_project_protocols.test", "graphql", "false"),
					resource.TestCheckResourceAttr("appwrite_project_protocols.test", "websocket", "false"),
					checkProtocols(t, map[string]bool{"rest": true, "graphql": false, "websocket": false}),
					checkGuestBlocked(t, "general_api_disabled", graphql),
					checkKeyBlocked(t, "general_api_disabled", graphql),
				),
			},
			{
				ResourceName:      "appwrite_project_protocols.test",
				ImportState:       true,
				ImportStateId:     os.Getenv("APPWRITE_PROJECT_ID"),
				ImportStateVerify: true,
			},
			{
				Config: `
resource "appwrite_project_protocols" "test" {
  graphql = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("appwrite_project_protocols.test", "rest"),
					resource.TestCheckNoResourceAttr("appwrite_project_protocols.test", "websocket"),
					checkProtocols(t, map[string]bool{"rest": true, "graphql": false, "websocket": true}),
				),
			},
			acceptance.ExpectEmptyPlanAfterApply(`
resource "appwrite_project_protocols" "test" {
  graphql = false
}
`),
		},
	})
}
