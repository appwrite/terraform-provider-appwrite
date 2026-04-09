package bucket_test

import (
	"fmt"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBucketResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBucketConfig("uploads", "Uploads"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_storage_bucket.test", "id", "uploads"),
					resource.TestCheckResourceAttr("appwrite_storage_bucket.test", "name", "Uploads"),
					resource.TestCheckResourceAttr("appwrite_storage_bucket.test", "enabled", "true"),
					resource.TestCheckResourceAttr("appwrite_storage_bucket.test", "encryption", "true"),
					resource.TestCheckResourceAttr("appwrite_storage_bucket.test", "antivirus", "true"),
					resource.TestCheckResourceAttr("appwrite_storage_bucket.test", "compression", "none"),
					resource.TestCheckResourceAttrSet("appwrite_storage_bucket.test", "created_at"),
				),
			},
			{
				ResourceName:      "appwrite_storage_bucket.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccBucketConfig("uploads", "User Uploads"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_storage_bucket.test", "name", "User Uploads"),
				),
			},
		},
	})
}

func TestAccBucketResource_with_options(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_storage_bucket" "test" {
  id                      = "images"
  name                    = "Images"
  maximum_file_size       = 10485760
  allowed_file_extensions = ["jpg", "png", "webp"]
  compression             = "gzip"
  transformations         = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_storage_bucket.test", "maximum_file_size", "10485760"),
					resource.TestCheckResourceAttr("appwrite_storage_bucket.test", "allowed_file_extensions.#", "3"),
					resource.TestCheckResourceAttr("appwrite_storage_bucket.test", "compression", "gzip"),
					resource.TestCheckResourceAttr("appwrite_storage_bucket.test", "transformations", "true"),
				),
			},
		},
	})
}

func testAccBucketConfig(id, name string) string {
	return fmt.Sprintf(`
resource "appwrite_storage_bucket" "test" {
  id   = %q
  name = %q
}
`, id, name)
}
