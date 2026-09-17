package provider_test

import (
	"strings"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
)

func newProvider() fwprovider.Provider {
	return provider.New("test")()
}

// Ephemeral resources and actions each reach the provider through their own
// interface, and a constructor that is written but never registered is invisible
// until someone writes the configuration for it. These walk what the provider
// actually advertises.
func TestEphemeralResourceSchemas(t *testing.T) {
	ctx := t.Context()

	p, ok := newProvider().(fwprovider.ProviderWithEphemeralResources)
	if !ok {
		t.Fatal("provider does not implement ProviderWithEphemeralResources")
	}

	want := map[string]bool{
		"appwrite_project_ephemeral_key": false,
		"appwrite_auth_session":          false,
		"appwrite_auth_jwt":              false,
	}

	for _, newResource := range p.EphemeralResources(ctx) {
		res := newResource()

		metaResp := &ephemeral.MetadataResponse{}
		res.Metadata(ctx, ephemeral.MetadataRequest{ProviderTypeName: "appwrite"}, metaResp)

		if _, expected := want[metaResp.TypeName]; !expected {
			t.Errorf("unexpected ephemeral resource %q", metaResp.TypeName)
			continue
		}
		want[metaResp.TypeName] = true

		schemaResp := &ephemeral.SchemaResponse{}
		res.Schema(ctx, ephemeral.SchemaRequest{}, schemaResp)
		if schemaResp.Diagnostics.HasError() {
			t.Errorf("%s: schema diagnostics: %v", metaResp.TypeName, schemaResp.Diagnostics)
		}
		if strings.TrimSpace(schemaResp.Schema.Description) == "" {
			t.Errorf("%s: schema has no description", metaResp.TypeName)
		}
		if _, cfg := res.(ephemeral.EphemeralResourceWithConfigure); !cfg {
			t.Errorf("%s: does not implement Configure, so it will never receive an API client", metaResp.TypeName)
		}
	}

	for name, seen := range want {
		if !seen {
			t.Errorf("ephemeral resource %q is not registered on the provider", name)
		}
	}
}

func TestActionSchemas(t *testing.T) {
	ctx := t.Context()

	p, ok := newProvider().(fwprovider.ProviderWithActions)
	if !ok {
		t.Fatal("provider does not implement ProviderWithActions")
	}

	want := map[string]bool{
		"appwrite_function_execution":             false,
		"appwrite_function_deployment_activation": false,
		"appwrite_postgresql_backup":              false,
		"appwrite_mysql_backup":                   false,
		"appwrite_mongo_backup":                   false,
	}

	for _, newAction := range p.Actions(ctx) {
		act := newAction()

		metaResp := &action.MetadataResponse{}
		act.Metadata(ctx, action.MetadataRequest{ProviderTypeName: "appwrite"}, metaResp)

		if _, expected := want[metaResp.TypeName]; !expected {
			t.Errorf("unexpected action %q", metaResp.TypeName)
			continue
		}
		want[metaResp.TypeName] = true

		schemaResp := &action.SchemaResponse{}
		act.Schema(ctx, action.SchemaRequest{}, schemaResp)
		if schemaResp.Diagnostics.HasError() {
			t.Errorf("%s: schema diagnostics: %v", metaResp.TypeName, schemaResp.Diagnostics)
		}
		if strings.TrimSpace(schemaResp.Schema.Description) == "" {
			t.Errorf("%s: schema has no description", metaResp.TypeName)
		}
		if _, cfg := act.(action.ActionWithConfigure); !cfg {
			t.Errorf("%s: does not implement Configure, so it will never receive an API client", metaResp.TypeName)
		}
	}

	for name, seen := range want {
		if !seen {
			t.Errorf("action %q is not registered on the provider", name)
		}
	}
}
