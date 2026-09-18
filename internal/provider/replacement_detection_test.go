package provider_test

import (
	"testing"

	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// TestReplacementDetectionStillRecognizesTheFramework is the canary for
// TestForceNewAttributesDocumentReplacement.
//
// That test decides whether an attribute forces replacement by reading what its
// plan modifier says about itself, because the framework builds RequiresReplace
// out of RequiresReplaceIf and the Go types are identical for both. The risk is
// silent: if a framework release rewords that sentence, detection returns false
// for everything, the documentation test passes with nothing to check, and 170
// attributes quietly stop being covered.
//
// This asserts the framework's own modifiers are still recognized, so a wording
// change fails here -- loudly, next to an explanation -- rather than disabling a
// check nobody is watching.
func TestReplacementDetectionStillRecognizesTheFramework(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		modifier            planmodifier.String
		wantForcesNew       bool
		wantOnlyIfConfigure bool
	}{
		"RequiresReplace": {
			modifier:      stringplanmodifier.RequiresReplace(),
			wantForcesNew: true,
		},
		"RequiresReplaceIfConfigured": {
			modifier:            stringplanmodifier.RequiresReplaceIfConfigured(),
			wantForcesNew:       true,
			wantOnlyIfConfigure: true,
		},
		"UseStateForUnknown does not force replacement": {
			modifier:      stringplanmodifier.UseStateForUnknown(),
			wantForcesNew: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			attr := rschema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{tc.modifier},
			}
			forcesNew, onlyIfConfigured := replacementBehavior(attr)

			if forcesNew != tc.wantForcesNew {
				t.Errorf("replacementBehavior detected forcesNew=%v, want %v.\n"+
					"The framework's plan modifier description has probably changed wording. "+
					"Update the matching in replacementBehavior, or force-new detection silently covers nothing.",
					forcesNew, tc.wantForcesNew)
			}
			if forcesNew && onlyIfConfigured != tc.wantOnlyIfConfigure {
				t.Errorf("onlyIfConfigured = %v, want %v", onlyIfConfigured, tc.wantOnlyIfConfigure)
			}
		})
	}
}

// TestForceNewDetectionFindsRealAttributes guards the same failure from the
// other side.
//
// The canary above proves the matcher recognizes a modifier handed to it
// directly. This proves the walker actually reaches the provider's attributes:
// a change to how schemas are traversed -- a new attribute type, a nesting shape
// the walker skips -- would drop the count towards zero while every assertion
// kept passing.
//
// The floor is deliberately well below the real number so ordinary work does not
// trip it. It exists to catch a collapse, not to pin a count.
func TestForceNewDetectionFindsRealAttributes(t *testing.T) {
	t.Parallel()

	const floor = 100

	attrs, err := walkResourceSchemas()
	if err != nil {
		t.Fatalf("walking schemas: %v", err)
	}

	var forceNew int
	for _, a := range attrs {
		if a.ForcesNewOnAny || a.ForcesNewIfSet {
			forceNew++
		}
	}

	if forceNew < floor {
		t.Errorf("found only %d attributes that force replacement, expected at least %d.\n"+
			"Either a great many were removed, or replacement detection has stopped working and "+
			"TestForceNewAttributesDocumentReplacement is now asserting nothing.", forceNew, floor)
	}
	if len(attrs) < 500 {
		t.Errorf("the schema walk found only %d attributes across all resources, which is too few to be right; "+
			"the walker is probably skipping a nesting shape", len(attrs))
	}
}
