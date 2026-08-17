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
		constructors[string(engine)+"_backup_storage"] = dedicated.NewBackupStorageResource(engine)
		constructors[string(engine)+"_branch"] = dedicated.NewBranchResource(engine)
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
		constructors[string(engine)+"_databases"] = dedicated.NewDatabasesDataSource(engine)
		constructors[string(engine)+"_specifications"] = dedicated.NewSpecificationsDataSource(engine)
		constructors[string(engine)+"_database_status"] = dedicated.NewStatusDataSource(engine)
		constructors[string(engine)+"_backups"] = dedicated.NewBackupsDataSource(engine)
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

// The PostgreSQL pooler ignores max_connections, so the attribute has to be
// computed-only there. That makes Terraform reject a configured value while
// planning; leaving it optional would instead force the provider to guess at
// apply time whether a known plan value came from the user or was carried
// forward from prior state by UseStateForUnknown, which it cannot tell apart.
func TestPoolerMaxConnectionsIsComputedOnlyOnPostgres(t *testing.T) {
	ctx := t.Context()

	for engine, wantConfigurable := range map[dedicated.Engine]bool{
		dedicated.EnginePostgresql: false,
		dedicated.EngineMysql:      true,
	} {
		schemaResp := &resource.SchemaResponse{}
		dedicated.NewPoolerResource(engine)().Schema(ctx, resource.SchemaRequest{}, schemaResp)

		attribute := schemaResp.Schema.Attributes["max_connections"]
		if got := attribute.IsOptional(); got != wantConfigurable {
			t.Errorf("%s max_connections IsOptional() = %v, want %v", engine, got, wantConfigurable)
		}
		if !attribute.IsComputed() {
			t.Errorf("%s max_connections should always be computed so it is read back from the server", engine)
		}
		if engine == dedicated.EnginePostgresql && !strings.Contains(attribute.GetDescription(), "Read-only on PostgreSQL") {
			t.Errorf("%s max_connections description should say it is read-only", engine)
		}
	}
}

// The other pooler settings must stay configurable on both engines; making them
// computed-only would silently ignore user configuration.
func TestPoolerWritableAttributesAreConfigurable(t *testing.T) {
	ctx := t.Context()

	writable := []string{"mode", "default_pool_size", "read_write_splitting",
		"pooler_cpu_request", "pooler_cpu_limit", "pooler_memory_request", "pooler_memory_limit"}

	for _, engine := range []dedicated.Engine{dedicated.EnginePostgresql, dedicated.EngineMysql} {
		schemaResp := &resource.SchemaResponse{}
		dedicated.NewPoolerResource(engine)().Schema(ctx, resource.SchemaRequest{}, schemaResp)

		for _, name := range writable {
			attribute, ok := schemaResp.Schema.Attributes[name]
			if !ok {
				t.Errorf("%s pooler is missing attribute %q", engine, name)
				continue
			}
			if !attribute.IsOptional() {
				t.Errorf("%s pooler attribute %q should be configurable", engine, name)
			}
		}
	}
}
