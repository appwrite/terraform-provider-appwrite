package provider_test

import (
	"strings"
	"testing"
)

func snapshotWith(attrs map[string]snapshotAttribute) providerSnapshot {
	return providerSnapshot{
		Resources: map[string]snapshotSchema{
			"appwrite_thing": {Attributes: attrs},
		},
		DataSources: map[string]snapshotSchema{},
		Ephemeral:   map[string]snapshotSchema{},
	}
}

// The asymmetry is the substance of the policy: loosening a constraint is safe,
// tightening one is not. Asserting it directly means the rule is tested rather
// than just described in a comment.
func TestBreakingChangesClassification(t *testing.T) {
	t.Parallel()

	optional := snapshotAttribute{Type: "tftypes.String", Optional: true}
	required := snapshotAttribute{Type: "tftypes.String", Required: true}
	computed := snapshotAttribute{Type: "tftypes.String", Computed: true}
	optionalComputed := snapshotAttribute{Type: "tftypes.String", Optional: true, Computed: true}
	sensitive := snapshotAttribute{Type: "tftypes.String", Optional: true, Sensitive: true}

	for name, tc := range map[string]struct {
		before, after map[string]snapshotAttribute
		wantKind      breakKind // empty means no break expected
	}{
		"nothing changed": {
			before: map[string]snapshotAttribute{"a": optional},
			after:  map[string]snapshotAttribute{"a": optional},
		},
		"attribute added is not breaking": {
			before: map[string]snapshotAttribute{"a": optional},
			after:  map[string]snapshotAttribute{"a": optional, "b": optional},
		},
		"required relaxed to optional is not breaking": {
			before: map[string]snapshotAttribute{"a": required},
			after:  map[string]snapshotAttribute{"a": optional},
		},
		"becoming computed as well is not breaking": {
			before: map[string]snapshotAttribute{"a": optional},
			after:  map[string]snapshotAttribute{"a": optionalComputed},
		},
		"gaining sensitivity is not breaking": {
			before: map[string]snapshotAttribute{"a": optional},
			after:  map[string]snapshotAttribute{"a": sensitive},
		},
		"attribute removed": {
			before:   map[string]snapshotAttribute{"a": optional, "b": optional},
			after:    map[string]snapshotAttribute{"a": optional},
			wantKind: breakRemoved,
		},
		"optional becomes required": {
			before:   map[string]snapshotAttribute{"a": optional},
			after:    map[string]snapshotAttribute{"a": required},
			wantKind: breakBecameRequired,
		},
		"type changed": {
			before:   map[string]snapshotAttribute{"a": optional},
			after:    map[string]snapshotAttribute{"a": {Type: "tftypes.Number", Optional: true}},
			wantKind: breakTypeChanged,
		},
		"computed dropped": {
			before:   map[string]snapshotAttribute{"a": computed},
			after:    map[string]snapshotAttribute{"a": {Type: "tftypes.String", Optional: true}},
			wantKind: breakComputedDropped,
		},
		"sensitivity dropped": {
			before:   map[string]snapshotAttribute{"a": sensitive},
			after:    map[string]snapshotAttribute{"a": optional},
			wantKind: breakSensitiveDropped,
		},
		"became write-only": {
			before:   map[string]snapshotAttribute{"a": optional},
			after:    map[string]snapshotAttribute{"a": {Type: "tftypes.String", Optional: true, WriteOnly: true}},
			wantKind: breakBecameWriteOnly,
		},
		// A nested attribute serializes as "object" whatever its shape, so
		// without the nesting mode recorded separately these two snapshots
		// compare equal and a change that rewrites every configuration using
		// the block passes the gate.
		"nested attribute changed from list to set": {
			before:   map[string]snapshotAttribute{"a": {Type: "object", Nesting: "list"}},
			after:    map[string]snapshotAttribute{"a": {Type: "object", Nesting: "set"}},
			wantKind: breakNestingChanged,
		},
		"block changed from single to list": {
			before:   map[string]snapshotAttribute{"a": {Type: "block", Nesting: "single"}},
			after:    map[string]snapshotAttribute{"a": {Type: "block", Nesting: "list"}},
			wantKind: breakNestingChanged,
		},
		"nesting unchanged is not breaking": {
			before: map[string]snapshotAttribute{"a": {Type: "block", Nesting: "single"}},
			after:  map[string]snapshotAttribute{"a": {Type: "block", Nesting: "single"}},
		},
		"no longer settable": {
			before:   map[string]snapshotAttribute{"a": optional},
			after:    map[string]snapshotAttribute{"a": {Type: "tftypes.String", Computed: true}},
			wantKind: breakNotSettable,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := breakingChanges(snapshotWith(tc.before), snapshotWith(tc.after))

			if tc.wantKind == "" {
				if len(got) > 0 {
					t.Fatalf("expected no breaking change, got %s", summarize(got))
				}
				return
			}
			if len(got) == 0 {
				t.Fatalf("expected a %s break, got none", tc.wantKind)
			}
			if !hasKind(got, tc.wantKind) {
				t.Errorf("expected a %s break, got %s", tc.wantKind, summarize(got))
			}
		})
	}
}

func TestBreakingChangesDetectsRemovedAndDowngradedTypes(t *testing.T) {
	t.Parallel()

	before := providerSnapshot{
		Resources:   map[string]snapshotSchema{"appwrite_gone": {Version: 2}, "appwrite_kept": {Version: 2}},
		DataSources: map[string]snapshotSchema{"appwrite_ds_gone": {}},
		Ephemeral:   map[string]snapshotSchema{"appwrite_eph_gone": {}},
	}
	after := providerSnapshot{
		Resources:   map[string]snapshotSchema{"appwrite_kept": {Version: 1}},
		DataSources: map[string]snapshotSchema{},
		Ephemeral:   map[string]snapshotSchema{},
	}

	got := breakingChanges(before, after)

	// One removal per kind of thing, plus the lowered version. A version going
	// backwards makes Terraform refuse to upgrade state it has already written,
	// which is as breaking as removing the resource outright.
	for _, target := range []string{"resource appwrite_gone", "data source appwrite_ds_gone", "ephemeral resource appwrite_eph_gone"} {
		if !hasKindForTarget(got, breakRemoved, target) {
			t.Errorf("expected %s to be reported as removed, got %s", target, summarize(got))
		}
	}
	if !hasKind(got, breakVersionLowered) {
		t.Errorf("expected a lowered schema version to be reported, got %s", summarize(got))
	}
}

func TestAddedSurfaceReportsNewThings(t *testing.T) {
	t.Parallel()

	before := providerSnapshot{
		Resources:   map[string]snapshotSchema{"appwrite_thing": {Attributes: map[string]snapshotAttribute{"a": {Type: "tftypes.String"}}}},
		DataSources: map[string]snapshotSchema{},
		Ephemeral:   map[string]snapshotSchema{},
	}
	after := providerSnapshot{
		Resources: map[string]snapshotSchema{
			"appwrite_thing": {Attributes: map[string]snapshotAttribute{"a": {Type: "tftypes.String"}, "b": {Type: "tftypes.String"}}},
			"appwrite_new":   {},
		},
		DataSources: map[string]snapshotSchema{},
		Ephemeral:   map[string]snapshotSchema{},
	}

	got := strings.Join(addedSurface(before, after), "\n")
	for _, want := range []string{`attribute "b" added`, "resource appwrite_new added"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in:\n%s", want, got)
		}
	}
	if len(breakingChanges(before, after)) != 0 {
		t.Error("additions must not be reported as breaking")
	}
}

func hasKind(changes []breakingChange, kind breakKind) bool {
	for _, c := range changes {
		if c.Kind == kind {
			return true
		}
	}
	return false
}

func hasKindForTarget(changes []breakingChange, kind breakKind, target string) bool {
	for _, c := range changes {
		if c.Kind == kind && c.Target == target {
			return true
		}
	}
	return false
}
