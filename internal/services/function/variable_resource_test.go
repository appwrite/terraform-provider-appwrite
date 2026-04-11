package function_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFunctionVariableResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_function" "test" {
  name    = "Variable Test Function"
  runtime = "node-22"
}

resource "appwrite_function_variable" "test" {
  function_id = appwrite_function.test.id
  key         = "API_URL"
  value       = "https://api.example.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_function_variable.test", "id"),
					resource.TestCheckResourceAttr("appwrite_function_variable.test", "key", "API_URL"),
					resource.TestCheckResourceAttrSet("appwrite_function_variable.test", "created_at"),
				),
			},
			{
				Config: `
resource "appwrite_function" "test" {
  name    = "Variable Test Function"
  runtime = "node-22"
}

resource "appwrite_function_variable" "test" {
  function_id = appwrite_function.test.id
  key         = "API_URL"
  value       = "https://api.example.com/v2"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_function_variable.test", "key", "API_URL"),
				),
			},
		},
	})
}
