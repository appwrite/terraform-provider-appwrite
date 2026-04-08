package column_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccColumnBaseConfig = `
resource "appwrite_database" "test" {
  id   = "blog"
  name = "Blog"
}

resource "appwrite_table" "test" {
  database_id = appwrite_database.test.id
  id          = "articles"
  name        = "Articles"
}
`

func TestAccColumnResource_varchar(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccColumnBaseConfig + `
resource "appwrite_column" "test" {
  database_id = appwrite_database.test.id
  table_id    = appwrite_table.test.id
  key         = "title"
  type        = "varchar"
  size        = 256
  required    = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_column.test", "key", "title"),
					resource.TestCheckResourceAttr("appwrite_column.test", "type", "varchar"),
					resource.TestCheckResourceAttr("appwrite_column.test", "size", "256"),
					resource.TestCheckResourceAttr("appwrite_column.test", "required", "true"),
					resource.TestCheckResourceAttr("appwrite_column.test", "array", "false"),
				),
			},
			{
				ResourceName:      "appwrite_column.test",
				ImportState:       true,
				ImportStateId:     "blog/articles/title",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccColumnResource_integer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccColumnBaseConfig + `
resource "appwrite_column" "test" {
  database_id = appwrite_database.test.id
  table_id    = appwrite_table.test.id
  key         = "word_count"
  type        = "integer"
  min         = 0
  max         = 100000
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_column.test", "key", "word_count"),
					resource.TestCheckResourceAttr("appwrite_column.test", "type", "integer"),
					resource.TestCheckResourceAttr("appwrite_column.test", "min", "0"),
					resource.TestCheckResourceAttr("appwrite_column.test", "max", "100000"),
					resource.TestCheckResourceAttr("appwrite_column.test", "required", "false"),
				),
			},
		},
	})
}

func TestAccColumnResource_float(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccColumnBaseConfig + `
resource "appwrite_column" "test" {
  database_id = appwrite_database.test.id
  table_id    = appwrite_table.test.id
  key         = "reading_time_minutes"
  type        = "float"
  float_min   = 0.5
  float_max   = 60.0
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_column.test", "key", "reading_time_minutes"),
					resource.TestCheckResourceAttr("appwrite_column.test", "type", "float"),
				),
			},
		},
	})
}

func TestAccColumnResource_boolean(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccColumnBaseConfig + `
resource "appwrite_column" "test" {
  database_id = appwrite_database.test.id
  table_id    = appwrite_table.test.id
  key         = "published"
  type        = "boolean"
  default     = "false"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_column.test", "key", "published"),
					resource.TestCheckResourceAttr("appwrite_column.test", "type", "boolean"),
					resource.TestCheckResourceAttr("appwrite_column.test", "required", "false"),
				),
			},
		},
	})
}

func TestAccColumnResource_enum(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccColumnBaseConfig + `
resource "appwrite_column" "test" {
  database_id = appwrite_database.test.id
  table_id    = appwrite_table.test.id
  key         = "category"
  type        = "enum"
  elements    = ["technology", "science", "design", "business"]
  default     = "technology"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_column.test", "key", "category"),
					resource.TestCheckResourceAttr("appwrite_column.test", "type", "enum"),
					resource.TestCheckResourceAttr("appwrite_column.test", "elements.#", "4"),
					resource.TestCheckResourceAttr("appwrite_column.test", "default", "technology"),
				),
			},
		},
	})
}

func TestAccColumnResource_email(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccColumnBaseConfig + `
resource "appwrite_column" "test" {
  database_id = appwrite_database.test.id
  table_id    = appwrite_table.test.id
  key         = "author_email"
  type        = "email"
  required    = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_column.test", "key", "author_email"),
					resource.TestCheckResourceAttr("appwrite_column.test", "type", "email"),
					resource.TestCheckResourceAttr("appwrite_column.test", "required", "true"),
				),
			},
		},
	})
}

func TestAccColumnResource_datetime(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccColumnBaseConfig + `
resource "appwrite_column" "test" {
  database_id = appwrite_database.test.id
  table_id    = appwrite_table.test.id
  key         = "published_at"
  type        = "datetime"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_column.test", "key", "published_at"),
					resource.TestCheckResourceAttr("appwrite_column.test", "type", "datetime"),
					resource.TestCheckResourceAttr("appwrite_column.test", "required", "false"),
				),
			},
		},
	})
}
