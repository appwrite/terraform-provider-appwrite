package file_test

import (
	"fmt"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccStorageFileResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_storage_bucket" "test" {
  name = "file-test-bucket"
}

resource "appwrite_storage_file" "test" {
  bucket_id = appwrite_storage_bucket.test.id
  name      = "test.txt"
  file_path = "testdata/test.txt"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_storage_file.test", "id"),
					resource.TestCheckResourceAttr("appwrite_storage_file.test", "name", "test.txt"),
					resource.TestCheckResourceAttrSet("appwrite_storage_file.test", "mime_type"),
					resource.TestCheckResourceAttrSet("appwrite_storage_file.test", "created_at"),
				),
			},
			{
				ResourceName: "appwrite_storage_file.test",
				ImportState:  true,
				// The bucket ID is server-generated, so the composite import ID
				// has to be built from state. Without this the framework passes
				// the bare file ID and the import is rejected.
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["appwrite_storage_file.test"]
					if !ok {
						return "", fmt.Errorf("resource appwrite_storage_file.test not found in state")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["bucket_id"], rs.Primary.Attributes["id"]), nil
				},
				ImportStateVerify: true,
				// file_path is a local path that the API has no concept of.
				ImportStateVerifyIgnore: []string{"file_path"},
			},
		},
	})
}
