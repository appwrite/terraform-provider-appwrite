package dedicated_test

import (
	"strings"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/services/dedicated"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var engines = []dedicated.Engine{
	dedicated.EnginePostgresql,
	dedicated.EngineMysql,
	dedicated.EngineMongo,
}

// The dedicated resources are built once and registered per engine, so a single
// schema mistake would reach every one of them. Validating each registered
// schema catches that at test time rather than at terraform plan.
func TestResourceSchemas(t *testing.T) {
	ctx := t.Context()

	constructors := map[string]func() resource.Resource{}
	for _, engine := range engines {
		constructors[string(engine)+"_database"] = dedicated.NewDatabaseResource(engine)
		constructors[string(engine)+"_backup_policy"] = dedicated.NewBackupPolicyResource(engine)
	}
	constructors["postgresql_pooler"] = dedicated.NewPoolerResource(dedicated.EnginePostgresql)
	constructors["mysql_pooler"] = dedicated.NewPoolerResource(dedicated.EngineMysql)
	constructors["postgresql_extension"] = dedicated.NewExtensionResource(dedicated.EnginePostgresql)

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

	constructors := map[string]func() datasource.DataSource{}
	for _, engine := range engines {
		constructors[string(engine)+"_database"] = dedicated.NewDatabaseDataSource(engine)
		constructors[string(engine)+"_specifications"] = dedicated.NewSpecificationsDataSource(engine)
	}
	constructors["postgresql_extensions"] = dedicated.NewExtensionsDataSource(dedicated.EnginePostgresql)

	for name, newDataSource := range constructors {
		t.Run(name, func(t *testing.T) {
			ds := newDataSource()

			metaResp := &datasource.MetadataResponse{}
			ds.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "appwrite"}, metaResp)
			if want := "appwrite_" + name; metaResp.TypeName != want {
				t.Errorf("type name = %q, want %q", metaResp.TypeName, want)
			}

			schemaResp := &datasource.SchemaResponse{}
			ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("schema returned errors: %v", schemaResp.Diagnostics)
			}
			if diags := schemaResp.Schema.ValidateImplementation(ctx); diags.HasError() {
				t.Errorf("invalid schema: %v", diags)
			}
		})
	}
}

// The database resource and data source are documented as mirroring each other,
// so every configurable attribute should be readable back through the data
// source. Drift between the two is easy to introduce and invisible until a user
// looks for a missing attribute.
func TestDataSourceCoversResourceAttributes(t *testing.T) {
	ctx := t.Context()

	resourceSchema := &resource.SchemaResponse{}
	dedicated.NewDatabaseResource(dedicated.EnginePostgresql)().Schema(ctx, resource.SchemaRequest{}, resourceSchema)

	dataSourceSchema := &datasource.SchemaResponse{}
	dedicated.NewDatabaseDataSource(dedicated.EnginePostgresql)().Schema(ctx, datasource.SchemaRequest{}, dataSourceSchema)

	for name := range resourceSchema.Schema.Attributes {
		if _, ok := dataSourceSchema.Schema.Attributes[name]; !ok {
			t.Errorf("attribute %q exists on the resource but not the data source", name)
		}
	}
}

func TestEngineLabels(t *testing.T) {
	for _, tc := range []struct {
		engine dedicated.Engine
		want   string
	}{
		{dedicated.EnginePostgresql, "PostgreSQL"},
		{dedicated.EngineMysql, "MySQL"},
		{dedicated.EngineMongo, "MongoDB"},
	} {
		if got := tc.engine.Label(); got != tc.want {
			t.Errorf("Label(%q) = %q, want %q", tc.engine, got, tc.want)
		}
	}
}

// The pooler's max_connections is read-only on PostgreSQL, and the schema
// description is where a user finds that out before an apply fails.
func TestPoolerMaxConnectionsDocumentedAsReadOnlyOnPostgres(t *testing.T) {
	ctx := t.Context()

	for engine, wantReadOnlyNote := range map[dedicated.Engine]bool{
		dedicated.EnginePostgresql: true,
		dedicated.EngineMysql:      false,
	} {
		schemaResp := &resource.SchemaResponse{}
		dedicated.NewPoolerResource(engine)().Schema(ctx, resource.SchemaRequest{}, schemaResp)

		description := schemaResp.Schema.Attributes["max_connections"].GetDescription()
		if got := strings.Contains(description, "Read-only on PostgreSQL"); got != wantReadOnlyNote {
			t.Errorf("%s max_connections read-only note = %v, want %v", engine, got, wantReadOnlyNote)
		}
	}
}
