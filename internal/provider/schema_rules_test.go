package provider_test

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/appwrite/terraform-provider-appwrite/internal/provider"
)

// TestForceNewAttributesDocumentReplacement asserts that every argument which
// cannot be changed in place says so.
//
// Before this test the provider had 170 such attributes and not one of them
// mentioned it, so the only way to discover that editing a field would recreate
// your database was to read a plan carefully -- or not read it carefully enough.
// The wording is fixed rather than free prose so that it is greppable, uniform
// across 46 resources, and cheap to assert.
func TestForceNewAttributesDocumentReplacement(t *testing.T) {
	t.Parallel()

	attrs, err := walkResourceSchemas()
	if err != nil {
		t.Fatalf("walking schemas: %v", err)
	}

	var undocumented []string
	for _, a := range attrs {
		if !a.ForcesNewOnAny && !a.ForcesNewIfSet {
			continue
		}
		if !strings.Contains(a.Description, common.ForcesReplacementNote) {
			undocumented = append(undocumented, a.Resource+"."+a.Path)
		}
	}

	if len(undocumented) > 0 {
		sort.Strings(undocumented)
		t.Errorf("%d attributes force replacement without saying so:\n  %s\n\n"+
			"Append common.ForcesReplacementNote to each description: %q\n"+
			"Note that a shared PlanModifiers variable hides RequiresReplace from a plain grep, which is why this "+
			"test reads the schema instead.",
			len(undocumented), strings.Join(undocumented, "\n  "), common.ForcesReplacementNote)
	}
}

// TestEveryAttributeHasADescription keeps undocumented attributes out.
//
// tfplugindocs happily generates a table row with an empty description, so an
// attribute with none is not a build failure -- it is a published reference page
// with a blank cell. Two timestamps were in exactly that state.
func TestEveryAttributeHasADescription(t *testing.T) {
	t.Parallel()

	attrs, err := walkResourceSchemas()
	if err != nil {
		t.Fatalf("walking schemas: %v", err)
	}

	var missing []string
	for _, a := range attrs {
		if strings.TrimSpace(a.Description) == "" {
			missing = append(missing, a.Resource+"."+a.Path)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("%d attributes have no description:\n  %s", len(missing), strings.Join(missing, "\n  "))
	}
}

// TestDescriptionsAreWellFormed applies the mechanically checkable half of a
// documentation standard: end with a full stop, do not start lower-case, and do
// not leak Go formatting artifacts.
//
// Deliberately not checking the stylistic rules AWS and azurerm also publish --
// no leading "The", booleans starting "Whether to" -- because retrofitting those
// across 695 attributes in the same change that introduces the check would bury
// everything else in this pull request. The check is here to be tightened.
func TestDescriptionsAreWellFormed(t *testing.T) {
	t.Parallel()

	attrs, err := walkResourceSchemas()
	if err != nil {
		t.Fatalf("walking schemas: %v", err)
	}

	for _, a := range attrs {
		desc := strings.TrimSpace(a.Description)
		if desc == "" {
			continue // reported by TestEveryAttributeHasADescription
		}
		name := a.Resource + "." + a.Path

		if !strings.HasSuffix(desc, ".") && !strings.HasSuffix(desc, "`") {
			t.Errorf("%s: description does not end with a full stop: %q", name, desc)
		}
		// A lower-case opener is a style slip, except where the word itself is
		// spelled that way -- "iOS bundle ID" is correct and rewording it to
		// satisfy a linter would make the documentation worse. Treating a
		// following capital as intentional covers that class without an
		// exceptions list.
		if runes := []rune(desc); runes[0] >= 'a' && runes[0] <= 'z' {
			secondIsUpper := len(runes) > 1 && runes[1] >= 'A' && runes[1] <= 'Z'
			if !secondIsUpper {
				t.Errorf("%s: description starts lower-case: %q", name, desc)
			}
		}
		if strings.Contains(desc, "  ") {
			t.Errorf("%s: description contains a double space, usually a missing space in a concatenation: %q", name, desc)
		}
		if strings.Contains(desc, "%!") || strings.Contains(desc, "%s") || strings.Contains(desc, "%d") {
			t.Errorf("%s: description contains an unformatted verb, so a Sprintf argument is missing: %q", name, desc)
		}
	}
}

// TestProviderSchemaIsValid runs the framework's own schema validation across
// every resource, data source and ephemeral resource at once.
//
// Terraform performs this at GetProviderSchema, which means a schema mistake --
// an attribute that is both Required and Computed, a nested object with no
// type, an invalid name -- surfaces as a plugin error on the user's first
// command rather than in CI. Calling the same entry point in a unit test moves
// that to the pull request, and it covers resources with no acceptance test at
// all.
func TestProviderSchemaIsValid(t *testing.T) {
	t.Parallel()

	server, err := providerserver.NewProtocol6WithError(provider.New("test")())()
	if err != nil {
		t.Fatalf("creating provider server: %v", err)
	}

	resp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %v", err)
	}

	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("schema error: %s: %s", d.Summary, d.Detail)
		}
	}

	// A provider that reports no resources would pass every assertion above
	// while being entirely broken, so assert the shape of the result too.
	if len(resp.ResourceSchemas) == 0 {
		t.Error("the provider reported no resource schemas")
	}
	if len(resp.DataSourceSchemas) == 0 {
		t.Error("the provider reported no data source schemas")
	}
	if len(resp.EphemeralResourceSchemas) == 0 {
		t.Error("the provider reported no ephemeral resource schemas")
	}
}
