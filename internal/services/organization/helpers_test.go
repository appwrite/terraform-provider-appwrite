package organization

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Appwrite matches an email to an account without regard to case, so a case
// change is the same membership. Replacing it would remove the member and add
// them back.
func TestEmailChangeRequiresReplace(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		state, plan types.String
		want        bool
	}{
		"case only":     {types.StringValue("jane@example.com"), types.StringValue("Jane@Example.com"), false},
		"other address": {types.StringValue("jane@example.com"), types.StringValue("joe@example.com"), true},
		"unknown":       {types.StringValue("jane@example.com"), types.StringUnknown(), true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var resp stringplanmodifier.RequiresReplaceIfFuncResponse
			emailChangeRequiresReplace(context.Background(), planmodifier.StringRequest{
				StateValue: tc.state,
				PlanValue:  tc.plan,
			}, &resp)
			if resp.RequiresReplace != tc.want {
				t.Errorf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.want)
			}
		})
	}
}

func TestSameRoles(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		a, b []string
		want bool
	}{
		"equal":           {[]string{"owner"}, []string{"owner"}, true},
		"reordered":       {[]string{"editor", "developer"}, []string{"developer", "editor"}, true},
		"different":       {[]string{"owner"}, []string{"developer"}, false},
		"subset":          {[]string{"owner"}, []string{"owner", "billing"}, false},
		"repeated":        {[]string{"owner", "owner"}, []string{"owner", "billing"}, false},
		"none from a nil": {nil, []string{}, true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := sameRoles(tc.a, tc.b); got != tc.want {
				t.Errorf("sameRoles(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
