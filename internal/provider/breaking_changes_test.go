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
		wantBreaking  bool
		wantContains  string
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
			before:       map[string]snapshotAttribute{"a": optional, "b": optional},
			after:        map[string]snapshotAttribute{"a": optional},
			wantBreaking: true,
			wantContains: `attribute "b" was removed`,
		},
		"optional becomes required": {
			before:       map[string]snapshotAttribute{"a": optional},
			after:        map[string]snapshotAttribute{"a": required},
			wantBreaking: true,
			wantContains: "became required",
		},
		"type changed": {
			before:       map[string]snapshotAttribute{"a": optional},
			after:        map[string]snapshotAttribute{"a": {Type: "tftypes.Number", Optional: true}},
			wantBreaking: true,
			wantContains: "changed type",
		},
		"computed dropped": {
			before:       map[string]snapshotAttribute{"a": computed},
			after:        map[string]snapshotAttribute{"a": {Type: "tftypes.String", Optional: true}},
			wantBreaking: true,
			wantContains: "no longer computed",
		},
		"sensitivity dropped": {
			before:       map[string]snapshotAttribute{"a": sensitive},
			after:        map[string]snapshotAttribute{"a": optional},
			wantBreaking: true,
			wantContains: "no longer sensitive",
		},
		"became write-only": {
			before:       map[string]snapshotAttribute{"a": optional},
			after:        map[string]snapshotAttribute{"a": {Type: "tftypes.String", Optional: true, WriteOnly: true}},
			wantBreaking: true,
			wantContains: "became write-only",
		},
		"no longer settable": {
			before:       map[string]snapshotAttribute{"a": optional},
			after:        map[string]snapshotAttribute{"a": {Type: "tftypes.String", Computed: true}},
			wantBreaking: true,
			wantContains: "no longer settable",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := breakingChanges(snapshotWith(tc.before), snapshotWith(tc.after))
			if tc.wantBreaking && len(got) == 0 {
				t.Fatalf("expected a breaking change, got none")
			}
			if !tc.wantBreaking && len(got) > 0 {
				t.Fatalf("expected no breaking change, got %v", got)
			}
			if tc.wantContains != "" && !strings.Contains(strings.Join(got, "\n"), tc.wantContains) {
				t.Errorf("expected a finding containing %q, got %v", tc.wantContains, got)
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

	got := strings.Join(breakingChanges(before, after), "\n")
	for _, want := range []string{
		"resource appwrite_gone was removed",
		"data source appwrite_ds_gone was removed",
		"ephemeral resource appwrite_eph_gone was removed",
		// A version going backwards makes Terraform refuse to upgrade state it
		// has already written, which is as breaking as removing the resource.
		"schema version went backwards",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in:\n%s", want, got)
		}
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
