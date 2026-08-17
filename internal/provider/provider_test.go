package provider_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// GetProviderSchema is where the framework rejects duplicate type names and
// invalid schemas. Booting the server here catches a bad registration without
// needing a Terraform CLI or an Appwrite instance.
func TestProviderSchema(t *testing.T) {
	ctx := t.Context()

	server, err := providerserver.NewProtocol6WithError(provider.New("test")())()
	if err != nil {
		t.Fatalf("creating provider server: %v", err)
	}

	resp, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("getting provider schema: %v", err)
	}
	for _, diagnostic := range resp.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("schema diagnostic: %s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}

	wantResources := []string{
		"appwrite_postgresql_database",
		"appwrite_mysql_database",
		"appwrite_mongo_database",
		"appwrite_postgresql_backup_policy",
		"appwrite_mysql_backup_policy",
		"appwrite_mongo_backup_policy",
		"appwrite_postgresql_pooler",
		"appwrite_mysql_pooler",
		"appwrite_postgresql_extension",
	}
	for _, name := range wantResources {
		if _, ok := resp.ResourceSchemas[name]; !ok {
			t.Errorf("resource %q is not registered", name)
		}
	}

	wantDataSources := []string{
		"appwrite_postgresql_database",
		"appwrite_mysql_database",
		"appwrite_mongo_database",
		"appwrite_postgresql_specifications",
		"appwrite_mysql_specifications",
		"appwrite_mongo_specifications",
		"appwrite_postgresql_extensions",
	}
	for _, name := range wantDataSources {
		if _, ok := resp.DataSourceSchemas[name]; !ok {
			t.Errorf("data source %q is not registered", name)
		}
	}

	// MongoDB has neither a pooler nor extensions, so registering those would
	// expose resources whose every apply fails against the API.
	unsupported := []string{
		"appwrite_mongo_pooler",
		"appwrite_mongo_extension",
		"appwrite_mysql_extension",
		"appwrite_mongo_extensions",
		"appwrite_mysql_extensions",
	}
	for _, name := range unsupported {
		if _, ok := resp.ResourceSchemas[name]; ok {
			t.Errorf("resource %q is registered but the engine does not support it", name)
		}
		if _, ok := resp.DataSourceSchemas[name]; ok {
			t.Errorf("data source %q is registered but the engine does not support it", name)
		}
	}
}

// Every registered type needs a generated docs page, or the Terraform Registry
// publishes it undocumented. Docs are generated from templates, so a new
// resource registered without its template silently ships without a page.
func TestRegisteredTypesHaveDocs(t *testing.T) {
	ctx := t.Context()

	server, err := providerserver.NewProtocol6WithError(provider.New("test")())()
	if err != nil {
		t.Fatalf("creating provider server: %v", err)
	}
	resp, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("getting provider schema: %v", err)
	}

	for _, tc := range []struct {
		kind    string
		dir     string
		schemas map[string]*tfprotov6.Schema
	}{
		{"resource", "resources", resp.ResourceSchemas},
		{"data source", "data-sources", resp.DataSourceSchemas},
	} {
		for name := range tc.schemas {
			page := filepath.Join("..", "..", "docs", tc.dir, strings.TrimPrefix(name, "appwrite_")+".md")
			if _, err := os.Stat(page); err != nil {
				t.Errorf("%s %q has no docs page at %s; run `make docs` after adding templates/%s/%s.md.tmpl",
					tc.kind, name, page, tc.dir, strings.TrimPrefix(name, "appwrite_"))
			}
		}
	}
}
