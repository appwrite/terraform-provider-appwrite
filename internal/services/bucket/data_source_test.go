package bucket_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBucketDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_storage_bucket" "test" {
  id   = "ds-test-bucket"
  name = "DS Test Bucket"
}

data "appwrite_storage_bucket" "test" {
  id = appwrite_storage_bucket.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.appwrite_storage_bucket.test", "name", "DS Test Bucket"),
					resource.TestCheckResourceAttr("data.appwrite_storage_bucket.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("data.appwrite_storage_bucket.test", "created_at"),
				),
			},
		},
	})
}
