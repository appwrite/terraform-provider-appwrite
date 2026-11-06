package project_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
)

// Three numbers are created together because Terraform creates them in
// parallel, and Appwrite loses one of two concurrent writes to a project's
// mock numbers. The count on the server is the check that the provider
// serializes them.
func TestAccProjectMockPhone_basic(t *testing.T) {
	config := func(otp string) string {
		return fmt.Sprintf(`
resource "appwrite_project_mock_phone" "test" {
  number = "+15555550100"
  otp    = %[1]q
}

resource "appwrite_project_mock_phone" "more" {
  count = 2

  number = "+1555555010${count.index + 1}"
  otp    = %[1]q
}
`, otp)
	}
	first := "appwrite_project_mock_phone.test"

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("123456"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(first, "id", "+15555550100"),
					resource.TestCheckResourceAttr(first, "otp", "123456"),
					resource.TestCheckResourceAttr(first, "project_id", os.Getenv("APPWRITE_PROJECT_ID")),
					resource.TestCheckResourceAttrSet(first, "created_at"),
					checkMockPhonesOnServer(t, 3, "123456"),
				),
			},
			{
				ResourceName:      first,
				ImportState:       true,
				ImportStateId:     os.Getenv("APPWRITE_PROJECT_ID") + "/+15555550100",
				ImportStateVerify: true,
			},
			{
				Config: config("654321"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(first, "otp", "654321"),
					checkMockPhonesOnServer(t, 3, "654321"),
				),
			},
			acceptance.ExpectEmptyPlanAfterApply(config("654321")),
		},
	})
}

func TestAccProjectMockPhone_invalid(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_project_mock_phone" "test" {
  number = "15555550100"
  otp    = "123456"
}
`,
				ExpectError: regexp.MustCompile(`must be an E.164 phone number`),
			},
			{
				Config: `
resource "appwrite_project_mock_phone" "test" {
  number = "+15555550100"
  otp    = "12345a"
}
`,
				ExpectError: regexp.MustCompile(`must be exactly six digits`),
			},
		},
	})
}

func checkMockPhonesOnServer(t *testing.T, want int, otp string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		list, err := appwrite.NewProject(acceptance.ProjectClient(t)).ListMockPhones()
		if err != nil {
			return err
		}
		if list.Total != want {
			return fmt.Errorf("%d mock numbers on the server, want %d", list.Total, want)
		}
		for _, mock := range list.MockNumbers {
			if mock.Otp != otp {
				return fmt.Errorf("mock number %s has code %s on the server, want %s", mock.Number, mock.Otp, otp)
			}
		}
		return nil
	}
}
