package provider_test

import (
	"os"
	"testing"
)

// TestSchemaHasNoBreakingChanges compares the provider's current schema against
// the committed baseline and fails on any change that would break an existing
// configuration or state.
//
// v2.1.0 has shipped, which means the breaking-change window for 2.x is closed:
// removing an attribute or making one required is now something users find out
// about when their plan fails. Nothing was watching for that, and a schema
// mistake is easy to make and almost invisible in a large diff.
//
// Refresh the baseline deliberately, after reading what changed:
//
//	APPWRITE_UPDATE_SCHEMA_BASELINE=1 go test ./internal/provider/ -run TestSchemaHasNoBreakingChanges
//
// Additions never fail this test, so a normal feature change needs the refresh
// but no judgement call. A reported break needs either a redesign or a major
// version; see contributing/breaking-changes.md.
func TestSchemaHasNoBreakingChanges(t *testing.T) {
	current, err := takeProviderSnapshot()
	if err != nil {
		t.Fatalf("snapshotting the current schema: %v", err)
	}

	if os.Getenv("APPWRITE_UPDATE_SCHEMA_BASELINE") != "" {
		encoded, err := marshalSnapshot(current)
		if err != nil {
			t.Fatalf("encoding the baseline: %v", err)
		}
		if err := os.WriteFile(baselinePath, encoded, 0o644); err != nil { //nolint:gosec // a committed baseline is not a secret
			t.Fatalf("writing the baseline: %v", err)
		}
		t.Log("schema baseline updated; review the diff before committing it")
		return
	}

	raw, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("reading the baseline at %s: %v\n"+
			"Generate it with APPWRITE_UPDATE_SCHEMA_BASELINE=1 go test ./internal/provider/", baselinePath, err)
	}

	var baseline providerSnapshot
	if err := unmarshalSnapshot(raw, &baseline); err != nil {
		t.Fatalf("parsing the baseline: %v", err)
	}

	if changes := breakingChanges(baseline, current); len(changes) > 0 {
		t.Errorf("the schema changed in %d backwards-incompatible way(s):\n%s\n\n"+
			"Breaking changes are not allowed within a major version; see contributing/breaking-changes.md.\n"+
			"If the change is deliberate and the major version is going with it, refresh the baseline:\n"+
			"    APPWRITE_UPDATE_SCHEMA_BASELINE=1 go test ./internal/provider/ -run TestSchemaHasNoBreakingChanges",
			len(changes), summarize(changes))
	}

	// The reverse direction is not a failure -- additions are fine -- but the
	// baseline still has to be refreshed, or it drifts until it stops being a
	// meaningful comparison.
	if additions := addedSurface(baseline, current); len(additions) > 0 {
		t.Errorf("the schema gained %d attribute(s) or type(s) the baseline does not know about:\n%s\n\n"+
			"These are not breaking. Refresh the baseline so it keeps covering the whole surface:\n"+
			"    APPWRITE_UPDATE_SCHEMA_BASELINE=1 go test ./internal/provider/ -run TestSchemaHasNoBreakingChanges",
			len(additions), summarizeStrings(additions))
	}
}
