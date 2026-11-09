package acceptance

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	sdkresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appwrite/terraform-provider-appwrite/internal/provider"
)

// TestDestroyCheckCoverage asserts that every managed resource the provider
// registers is either verifiable after destroy or has a recorded reason it is
// not.
//
// This is the ratchet. Adding a resource without thinking about its destroy
// check fails here rather than shipping a test suite that appears to cover it.
// Fixing the failure means writing the check, or writing down why not -- both
// acceptable, silence is not.
func TestDestroyCheckCoverage(t *testing.T) {
	t.Parallel()

	registered := registeredResourceTypes(t)

	var missing []string
	for _, typeName := range registered {
		_, checked := destroyChecks[typeName]
		_, reset := resetChecks[typeName]
		_, excused := destroyCheckGaps[typeName]
		if !checked && !reset && !excused {
			missing = append(missing, typeName)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("no destroy check and no recorded reason for %v\n"+
			"Add a check to destroyChecks (or resetChecks, for a settings resource) in destroy.go, or record why it cannot have one in destroyCheckGaps.", missing)
	}

	// The other direction matters just as much. A stale entry naming a resource
	// that no longer exists, or one whose type name was renamed, quietly stops
	// covering anything.
	known := make(map[string]bool, len(registered))
	for _, typeName := range registered {
		known[typeName] = true
	}
	for typeName := range destroyChecks {
		if !known[typeName] {
			t.Errorf("destroyChecks has an entry for %q, which the provider does not register", typeName)
		}
	}
	for typeName := range resetChecks {
		if !known[typeName] {
			t.Errorf("resetChecks has an entry for %q, which the provider does not register", typeName)
		}
		if _, both := destroyChecks[typeName]; both {
			t.Errorf("%q has both a destroy check and a reset check; a resource is either deleted or reset", typeName)
		}
	}
	for typeName := range destroyCheckGaps {
		if !known[typeName] {
			t.Errorf("destroyCheckGaps has an entry for %q, which the provider does not register", typeName)
		}
	}
}

func registeredResourceTypes(t *testing.T) []string {
	t.Helper()

	ctx := context.Background()
	factories := provider.New("test")().Resources(ctx)
	types := make([]string, 0, len(factories))
	for _, newResource := range factories {
		var resp resource.MetadataResponse
		newResource().Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "appwrite"}, &resp)
		if resp.TypeName == "" {
			t.Fatalf("a resource returned an empty type name from Metadata")
		}
		types = append(types, resp.TypeName)
	}
	sort.Strings(types)
	return types
}

// TestCheckDestroyIgnoresForeignResources pins the behavior that broke the
// ephemeral tests: the echo provider leaves a resource in state that has no
// server side, and reporting it as unchecked failed every test using it.
func TestCheckDestroyIgnoresForeignResources(t *testing.T) {
	t.Setenv(sdkresource.EnvTfAcc, "1")

	state := &terraform.State{
		Modules: []*terraform.ModuleState{{
			Path: []string{"root"},
			Resources: map[string]*terraform.ResourceState{
				"echo.test": {
					Type:    "echo",
					Primary: &terraform.InstanceState{ID: "echo", Attributes: map[string]string{}},
				},
			},
		}},
	}

	if err := CheckDestroy(t)(state); err != nil {
		t.Fatalf("a resource from another provider was reported: %v", err)
	}
}

// An OAuth2 provider cannot be deleted, only reset, so its destroy check
// passes on a reset provider and fails on one still holding settings.
func TestOAuthProviderReset(t *testing.T) {
	t.Parallel()

	for body, wantReset := range map[string]bool{
		`{"$id":"github","enabled":false,"clientId":"","clientSecret":""}`:                                    true,
		`{"$id":"google","enabled":false,"clientId":"","prompt":["consent"],"nativeEnabled":false}`:           true,
		`{"$id":"oidc","enabled":false,"clientId":"","wellKnownURL":"","prompt":[],"maxAge":600}`:             true,
		`{"$id":"github","enabled":true,"clientId":"","clientSecret":""}`:                                     false,
		`{"$id":"github","enabled":false,"clientId":"abc","clientSecret":""}`:                                 false,
		`{"$id":"keycloak","enabled":false,"clientId":"","endpoint":"","realmName":"main"}`:                   false,
		`{"$id":"apple","enabled":false,"serviceId":"","nativeEnabled":true,"nativeClientIds":["com.x.app"]}`: false,
	} {
		err := oauthProviderReset([]byte(body))
		if got := errors.Is(err, errReset); got != wantReset {
			t.Errorf("%s: reset = %v (%v), want %v", body, got, err, wantReset)
		}
	}
}
