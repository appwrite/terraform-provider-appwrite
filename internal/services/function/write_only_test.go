package function_test

import (
	"fmt"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// A non-secret variable has its value returned by the API on both create and
// read, so a mapper that refreshes `value` unconditionally would write a
// value_wo secret straight into state. This is the case that catches that: the
// variable is deliberately not marked secret, which is when the API hands the
// value back.
func TestAccFunctionVariableResource_writeOnlyValueStaysOutOfState(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck: func() { acceptance.PreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFunctionVariableWriteOnlyConfig("wo-value-one", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("appwrite_function_variable.wo", "value"),
					resource.TestCheckNoResourceAttr("appwrite_function_variable.wo", "value_wo"),
					resource.TestCheckResourceAttr("appwrite_function_variable.wo", "value_wo_version", "1"),
				),
			},
			{
				// A refresh is where the API's returned value would leak in, so
				// the second step matters as much as the first.
				Config: testAccFunctionVariableWriteOnlyConfig("wo-value-two", 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("appwrite_function_variable.wo", "value"),
					resource.TestCheckResourceAttr("appwrite_function_variable.wo", "value_wo_version", "2"),
				),
			},
		},
	})
}

func testAccFunctionVariableWriteOnlyConfig(value string, version int) string {
	return fmt.Sprintf(`
resource "appwrite_function" "wo" {
  id      = "wo-var-fn"
  name    = "Write-only variable test"
  runtime = "node-22"
}

resource "appwrite_function_variable" "wo" {
  function_id     = appwrite_function.wo.id
  key             = "PLAIN"
  secret          = false
  value_wo        = %q
  value_wo_version = %d
}
`, value, version)
}
