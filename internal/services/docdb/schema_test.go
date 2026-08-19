package docdb_test

import (
	"strings"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/services/docdb"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var products = []docdb.Product{docdb.ProductDocumentsDB, docdb.ProductVectorsDB}

// Each resource is written once and registered per product, so one schema
// mistake reaches both. Validating every registered schema catches that here
// rather than at terraform plan.
func TestResourceSchemas(t *testing.T) {
	ctx := t.Context()

	constructors := map[string]func() resource.Resource{}
	for _, product := range products {
		constructors[string(product)] = docdb.NewDatabaseResource(product)
		constructors[string(product)+"_collection"] = docdb.NewCollectionResource(product)
		constructors[string(product)+"_index"] = docdb.NewIndexResource(product)
		constructors[string(product)+"_document"] = docdb.NewDocumentResource(product)
	}

	for name, newResource := range constructors {
		t.Run(name, func(t *testing.T) {
			res := newResource()

			metaResp := &resource.MetadataResponse{}
			res.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "appwrite"}, metaResp)
			if want := "appwrite_" + name; metaResp.TypeName != want {
				t.Errorf("type name = %q, want %q", metaResp.TypeName, want)
			}

			schemaResp := &resource.SchemaResponse{}
			res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("schema returned errors: %v", schemaResp.Diagnostics)
			}
			if diags := schemaResp.Schema.ValidateImplementation(ctx); diags.HasError() {
				t.Errorf("invalid schema: %v", diags)
			}
		})
	}
}

func TestDataSourceSchemas(t *testing.T) {
	ctx := t.Context()

	for _, product := range products {
		t.Run(string(product), func(t *testing.T) {
			ds := docdb.NewDatabaseDataSource(product)()

			metaResp := &datasource.MetadataResponse{}
			ds.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "appwrite"}, metaResp)
			if want := "appwrite_" + string(product); metaResp.TypeName != want {
				t.Errorf("type name = %q, want %q", metaResp.TypeName, want)
			}

			schemaResp := &datasource.SchemaResponse{}
			ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
			if diags := schemaResp.Schema.ValidateImplementation(ctx); diags.HasError() {
				t.Errorf("invalid schema: %v", diags)
			}
		})
	}
}

// Only VectorsDB collections take an embedding dimension. Making it required
// there and computed-only on DocumentsDB means Terraform rejects a configured
// dimension on DocumentsDB while planning, instead of the provider silently
// dropping a value the route never accepts.
func TestCollectionDimensionIsProductSpecific(t *testing.T) {
	ctx := t.Context()

	for product, wantConfigurable := range map[docdb.Product]bool{
		docdb.ProductVectorsDB:   true,
		docdb.ProductDocumentsDB: false,
	} {
		schemaResp := &resource.SchemaResponse{}
		docdb.NewCollectionResource(product)().Schema(ctx, resource.SchemaRequest{}, schemaResp)

		attribute, ok := schemaResp.Schema.Attributes["dimension"]
		if !ok {
			t.Fatalf("%s collection has no dimension attribute", product)
		}
		if got := attribute.IsRequired(); got != wantConfigurable {
			t.Errorf("%s dimension IsRequired() = %v, want %v", product, got, wantConfigurable)
		}
		if got := attribute.IsComputed(); got == wantConfigurable {
			t.Errorf("%s dimension IsComputed() = %v, want %v", product, got, !wantConfigurable)
		}
	}
}

func TestProductLabels(t *testing.T) {
	for product, want := range map[docdb.Product]string{
		docdb.ProductDocumentsDB: "DocumentsDB",
		docdb.ProductVectorsDB:   "VectorsDB",
	} {
		if got := product.Label(); got != want {
			t.Errorf("Label(%q) = %q, want %q", product, got, want)
		}
	}
}

// Attributes can only be declared when a collection is created, and only on
// DocumentsDB. Exposing them as configurable on VectorsDB would accept input
// the route never sends; leaving out RequiresReplace would silently drop a
// changed definition, since there is no route to alter one in place.
func TestCollectionAttributesAreCreateOnlyAndProductSpecific(t *testing.T) {
	ctx := t.Context()

	for product, wantConfigurable := range map[docdb.Product]bool{
		docdb.ProductDocumentsDB: true,
		docdb.ProductVectorsDB:   false,
	} {
		schemaResp := &resource.SchemaResponse{}
		docdb.NewCollectionResource(product)().Schema(ctx, resource.SchemaRequest{}, schemaResp)

		attribute, ok := schemaResp.Schema.Attributes["attributes"]
		if !ok {
			t.Fatalf("%s collection has no attributes field", product)
		}
		if got := attribute.IsOptional(); got != wantConfigurable {
			t.Errorf("%s attributes IsOptional() = %v, want %v", product, got, wantConfigurable)
		}
		if got := attribute.IsComputed(); got == wantConfigurable {
			t.Errorf("%s attributes IsComputed() = %v, want %v", product, got, !wantConfigurable)
		}
		if wantConfigurable && !strings.Contains(attribute.GetDescription(), "replaces the collection") {
			t.Errorf("%s attributes description should say it is create-only", product)
		}
	}
}

// An index exists remotely the moment CreateIndex returns, but the build is
// asynchronous. Create must therefore write the index's identity to state
// before waiting: a wait that times out, or reports a stuck or failed build,
// would otherwise leave Terraform with no record of an index that exists.
// Every attribute making up that identity has to be settable independently of
// the wait, so none of them may be read-only.
func TestIndexIdentityAttributesAreWritableBeforeTheWait(t *testing.T) {
	ctx := t.Context()

	for _, product := range products {
		schemaResp := &resource.SchemaResponse{}
		docdb.NewIndexResource(product)().Schema(ctx, resource.SchemaRequest{}, schemaResp)

		for _, name := range []string{"database_id", "collection_id", "key"} {
			attribute, ok := schemaResp.Schema.Attributes[name]
			if !ok {
				t.Errorf("%s index is missing %q", product, name)
				continue
			}
			if !attribute.IsRequired() {
				t.Errorf("%s index %q should be required so identity is known before the wait", product, name)
			}
		}
		if id, ok := schemaResp.Schema.Attributes["id"]; !ok || !id.IsComputed() {
			t.Errorf("%s index id should be computed", product)
		}
	}
}
