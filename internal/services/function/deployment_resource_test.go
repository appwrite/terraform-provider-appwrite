package function_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFunctionDeploymentResource_template(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_function" "deploy_test" {
  name    = "Deployment Test"
  runtime = "node-22"
}

resource "appwrite_function_deployment" "test" {
  function_id    = appwrite_function.deploy_test.id
  source_type    = "template"
  repository     = "templates-for-node-22"
  owner          = "appwrite"
  root_directory = "starter"
  type           = "branch"
  reference      = "main"
  activate       = true
  wait_for_ready = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_function_deployment.test", "id"),
					resource.TestCheckResourceAttr("appwrite_function_deployment.test", "source_type", "template"),
					resource.TestCheckResourceAttr("appwrite_function_deployment.test", "status", "ready"),
					resource.TestCheckResourceAttrSet("appwrite_function_deployment.test", "created_at"),
				),
			},
			{
				ResourceName:            "appwrite_function_deployment.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"source_type", "repository", "owner", "root_directory", "type", "reference", "activate", "wait_for_ready"},
			},
		},
	})
}
