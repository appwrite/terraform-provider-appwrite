package column_test

import (
	"fmt"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccColumnBaseConfig = `
resource "appwrite_tablesdb" "test" {
  id   = "blog"
  name = "Blog"
}

resource "appwrite_tablesdb_table" "test" {
  database_id = appwrite_tablesdb.test.id
  id          = "articles"
  name        = "Articles"
}
`

func TestAccColumnResource_varchar(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccColumnBaseConfig + `
resource "appwrite_tablesdb_column" "test" {
  database_id = appwrite_tablesdb.test.id
  table_id    = appwrite_tablesdb_table.test.id
  key         = "title"
  type        = "varchar"
  size        = 256
  required    = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "key", "title"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "type", "varchar"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "size", "256"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "required", "true"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "array", "false"),
				),
			},
			{
				ResourceName:                         "appwrite_tablesdb_column.test",
				ImportState:                          true,
				ImportStateId:                        "blog/articles/title",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "key",
				ImportStateVerifyIgnore:              []string{"updated_at"},
			},
		},
	})
}

func TestAccColumnResource_string(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccColumnBaseConfig + `
resource "appwrite_tablesdb_column" "test" {
  database_id = appwrite_tablesdb.test.id
  table_id    = appwrite_tablesdb_table.test.id
  key         = "slug"
  type        = "string"
  size        = 36
  required    = true
}

# A sized string column must be indexable (the original bug materialized the
# column as unbounded TEXT, which Appwrite refuses to index).
resource "appwrite_tablesdb_index" "test" {
  database_id = appwrite_tablesdb.test.id
  table_id    = appwrite_tablesdb_table.test.id
  key         = "idx_slug"
  type        = "key"
  columns     = ["slug"]
  orders      = ["ASC"]

  depends_on = [appwrite_tablesdb_column.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "key", "slug"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "type", "string"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "size", "36"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "required", "true"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "array", "false"),
				),
			},
			{
				ResourceName:                         "appwrite_tablesdb_column.test",
				ImportState:                          true,
				ImportStateId:                        "blog/articles/slug",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "key",
				ImportStateVerifyIgnore:              []string{"updated_at"},
			},
		},
	})
}

func TestAccColumnResource_integer(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccColumnBaseConfig + `
resource "appwrite_tablesdb_column" "test" {
  database_id = appwrite_tablesdb.test.id
  table_id    = appwrite_tablesdb_table.test.id
  key         = "word_count"
  type        = "integer"
  min         = 0
  max         = 100000
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "key", "word_count"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "type", "integer"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "min", "0"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "max", "100000"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "required", "false"),
				),
			},
		},
	})
}

func TestAccColumnResource_float(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccColumnBaseConfig + `
resource "appwrite_tablesdb_column" "test" {
  database_id = appwrite_tablesdb.test.id
  table_id    = appwrite_tablesdb_table.test.id
  key         = "reading_time_minutes"
  type        = "float"
  float_min   = 0.5
  float_max   = 60.0
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "key", "reading_time_minutes"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "type", "float"),
				),
			},
		},
	})
}

// Appwrite fills an omitted bound with the widest value the type allows and
// returns it. Each step must apply cleanly, and the harness's no-replace guard
// fails the test if the column is tainted and planned for replacement.
func TestAccColumnResource_unbounded(t *testing.T) {
	const columns = `
resource "appwrite_tablesdb_column" "count" {
  database_id = appwrite_tablesdb.test.id
  table_id    = appwrite_tablesdb_table.test.id
  key         = "view_count"
  type        = "integer"
  %s
}

resource "appwrite_tablesdb_column" "ratio" {
  database_id = appwrite_tablesdb.test.id
  table_id    = appwrite_tablesdb_table.test.id
  key         = "ratio"
  type        = "float"
}
`
	unbounded := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckNoResourceAttr("appwrite_tablesdb_column.count", "min"),
		resource.TestCheckNoResourceAttr("appwrite_tablesdb_column.count", "max"),
		resource.TestCheckNoResourceAttr("appwrite_tablesdb_column.ratio", "float_min"),
		resource.TestCheckNoResourceAttr("appwrite_tablesdb_column.ratio", "float_max"),
	)

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccColumnBaseConfig + fmt.Sprintf(columns, ""),
				Check:  unbounded,
			},
			{
				Config: testAccColumnBaseConfig + fmt.Sprintf(columns, "max = 1000"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("appwrite_tablesdb_column.count", "min"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.count", "max", "1000"),
				),
			},
			// Removing the bound must reset it on the server: an update that
			// omits a bound keeps the stored one.
			{
				Config: testAccColumnBaseConfig + fmt.Sprintf(columns, ""),
				Check:  unbounded,
			},
			{
				ResourceName:                         "appwrite_tablesdb_column.count",
				ImportState:                          true,
				ImportStateId:                        "blog/articles/view_count",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "key",
				ImportStateVerifyIgnore:              []string{"updated_at"},
			},
		},
	})
}

func TestAccColumnResource_boolean(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccColumnBaseConfig + `
resource "appwrite_tablesdb_column" "test" {
  database_id = appwrite_tablesdb.test.id
  table_id    = appwrite_tablesdb_table.test.id
  key         = "published"
  type        = "boolean"
  default     = "false"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "key", "published"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "type", "boolean"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "required", "false"),
				),
			},
		},
	})
}

func TestAccColumnResource_enum(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccColumnBaseConfig + `
resource "appwrite_tablesdb_column" "test" {
  database_id = appwrite_tablesdb.test.id
  table_id    = appwrite_tablesdb_table.test.id
  key         = "category"
  type        = "enum"
  elements    = ["technology", "science", "design", "business"]
  default     = "technology"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "key", "category"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "type", "enum"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "elements.#", "4"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "default", "technology"),
				),
			},
		},
	})
}

func TestAccColumnResource_email(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccColumnBaseConfig + `
resource "appwrite_tablesdb_column" "test" {
  database_id = appwrite_tablesdb.test.id
  table_id    = appwrite_tablesdb_table.test.id
  key         = "author_email"
  type        = "email"
  required    = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "key", "author_email"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "type", "email"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "required", "true"),
				),
			},
		},
	})
}

func TestAccColumnResource_datetime(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccColumnBaseConfig + `
resource "appwrite_tablesdb_column" "test" {
  database_id = appwrite_tablesdb.test.id
  table_id    = appwrite_tablesdb_table.test.id
  key         = "published_at"
  type        = "datetime"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "key", "published_at"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "type", "datetime"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_column.test", "required", "false"),
				),
			},
		},
	})
}
