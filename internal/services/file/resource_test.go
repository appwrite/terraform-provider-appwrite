package file_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
				ResourceName:            "appwrite_storage_file.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"file_path"},
			},
		},
	})
}
