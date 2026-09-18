package acceptance

import (
	"context"
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
		_, excused := destroyCheckGaps[typeName]
		if !checked && !excused {
			missing = append(missing, typeName)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("no destroy check and no recorded reason for %v\n"+
			"Add a check to destroyChecks in destroy.go, or record why it cannot have one in destroyCheckGaps.", missing)
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
