package team

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestCanonicalPrefs(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name, in, want string
	}{
		{name: "empty", in: `{}`, want: `{}`},
		{name: "keys sorted", in: `{"b":1,"a":2}`, want: `{"a":2,"b":1}`},
		{name: "nested keys sorted", in: `{"x":{"b":1,"a":[{"d":1,"c":2}]}}`, want: `{"x":{"a":[{"c":2,"d":1}],"b":1}}`},
		{name: "integral float becomes integer", in: `{"a":1.0,"b":-2.00,"c":1e3}`, want: `{"a":1,"b":-2,"c":1000}`},
		{name: "fraction kept", in: `{"a":1.5}`, want: `{"a":1.5}`},
		// 2^53 + 1 has no exact float64; going through float64 would store
		// ...992.
		{name: "large integer exact", in: `{"a":9007199254740993}`, want: `{"a":9007199254740993}`},
		// Beyond int64 the server stores a float, and both spellings must
		// compare equal.
		{name: "beyond int64", in: `{"a":12345678901234567890}`, want: `{"a":12345678901234567000}`},
		{name: "beyond int64 as server returns it", in: `{"a":1.2345678901234567e+19}`, want: `{"a":12345678901234567000}`},
		// jsonencode escapes <, > and & too, so its output is already
		// canonical.
		{name: "html escaped as jsonencode does", in: `{"a":"<b>&"}`, want: "{\"a\":\"\\u003cb\\u003e\\u0026\"}"},
		{name: "whitespace", in: " {\n \"a\" : null } ", want: `{"a":null}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := canonicalPrefs(tc.in)
			if err != nil {
				t.Fatalf("canonicalPrefs(%s): %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("canonicalPrefs(%s) = %s, want %s", tc.in, got, tc.want)
			}
		})
	}
}

func TestCanonicalPrefsRejects(t *testing.T) {
	t.Parallel()
	for _, in := range []string{``, `[]`, `[1]`, `"x"`, `1`, `true`, `null`, `{`, `{} {}`, `{"a":1}x`} {
		if got, err := canonicalPrefs(in); err == nil {
			t.Errorf("canonicalPrefs(%q) = %s, want an error", in, got)
		}
	}
}

func TestPrefsForState(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		prior  types.String
		actual string
		want   string
	}{
		{name: "no prior", prior: types.StringNull(), actual: `{"b":1,"a":2}`, want: `{"a":2,"b":1}`},
		{name: "unknown prior", prior: types.StringUnknown(), actual: `{}`, want: `{}`},
		// jsonencode's spelling is kept when it means the same object.
		{name: "prior spelling kept", prior: types.StringValue(`{"a":2.0, "b":1}`), actual: `{"b":1,"a":2}`, want: `{"a":2.0, "b":1}`},
		{name: "drift replaces prior", prior: types.StringValue(`{"a":1}`), actual: `{"a":2}`, want: `{"a":2}`},
		{name: "unparseable prior replaced", prior: types.StringValue(`nope`), actual: `{"a":2}`, want: `{"a":2}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := prefsForState(tc.prior, tc.actual)
			if err != nil {
				t.Fatal(err)
			}
			if got.ValueString() != tc.want {
				t.Errorf("got %s, want %s", got.ValueString(), tc.want)
			}
		})
	}

	if _, err := prefsForState(types.StringNull(), `[1]`); err == nil {
		t.Error("a non-object from the server should be an error")
	}
}

func TestRawPrefs(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name, body, want string
	}{
		{name: "object", body: `{"$id":"t","prefs":{"a":1}}`, want: `{"a":1}`},
		{name: "missing", body: `{"$id":"t"}`, want: `{}`},
		{name: "null", body: `{"prefs":null}`, want: `{}`},
		// PHP encodes an empty associative array as [].
		{name: "empty array", body: `{"prefs":[]}`, want: `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := rawPrefs(func(v interface{}) error { return jsonUnmarshal(tc.body, v) })
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestPrefsValidator(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		value   types.String
		wantErr bool
	}{
		{value: types.StringNull()},
		{value: types.StringUnknown()},
		{value: types.StringValue(`{}`)},
		{value: types.StringValue(`{"a":[1,2]}`)},
		{value: types.StringValue(`[]`), wantErr: true},
		{value: types.StringValue(`"x"`), wantErr: true},
		{value: types.StringValue(`{`), wantErr: true},
	} {
		resp := &validator.StringResponse{}
		prefsValidator{}.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("prefs"), ConfigValue: tc.value}, resp)
		if resp.Diagnostics.HasError() != tc.wantErr {
			t.Errorf("%s: error = %v, want %v (%v)", tc.value, resp.Diagnostics.HasError(), tc.wantErr, resp.Diagnostics)
		}
	}
}

func TestRoleKeyPattern(t *testing.T) {
	t.Parallel()

	long := "a"
	for len(long) < 81 {
		long += "b"
	}
	for _, role := range []string{"owner", "a", "A.b-c_d", "0lead", long} {
		if !roleKeyPattern.MatchString(role) {
			t.Errorf("%q should be accepted", role)
		}
	}
	// The server rejects all of these.
	for _, role := range []string{"", "_a", ".a", "-a", "has space", "a:b", "team/owner", long + "c"} {
		if roleKeyPattern.MatchString(role) {
			t.Errorf("%q should be rejected", role)
		}
	}
}

func TestUniqueRoles(t *testing.T) {
	t.Parallel()
	got := uniqueRoles([]string{"zeta", "alpha", "zeta"})
	if len(got) != 2 || got[0] != "zeta" || got[1] != "alpha" {
		t.Errorf("got %v", got)
	}
	if got := uniqueRoles(nil); got == nil || len(got) != 0 {
		t.Errorf("nil roles should give an empty, non-nil slice, got %#v", got)
	}
}

func TestSameRoles(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a, b []string
		want bool
	}{
		{[]string{"dev", "lead"}, []string{"lead", "dev"}, true},
		{[]string{"dev", "dev"}, []string{"dev"}, true},
		{nil, []string{}, true},
		{[]string{"dev"}, []string{"lead"}, false},
		{[]string{"dev"}, []string{"dev", "lead"}, false},
	} {
		if got := sameRoles(tc.a, tc.b); got != tc.want {
			t.Errorf("sameRoles(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

// TestRequiresReplaceUnlessAdopting covers a membership that was not
// imported: changing or adding email replaces it. The framework's private
// state cannot be constructed outside it, so the imported case is covered by
// TestAccTeamMembership_import instead.
func TestRequiresReplaceUnlessAdopting(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var s resource.SchemaResponse
	NewMembershipResource().Schema(ctx, resource.SchemaRequest{}, &s)

	for _, tc := range []struct {
		name  string
		state types.String
		plan  types.String
	}{
		{name: "created by user_id, email added", state: types.StringNull(), plan: types.StringValue("a@example.com")},
		{name: "created by email, changed", state: types.StringValue("a@example.com"), plan: types.StringValue("b@example.com")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := planmodifier.StringRequest{
				Path:        path.Root("email"),
				StateValue:  tc.state,
				PlanValue:   tc.plan,
				ConfigValue: tc.plan,
				State:       tfsdk.State{Schema: s.Schema, Raw: objectWithEmail(t, s, tc.state)},
				Plan:        tfsdk.Plan{Schema: s.Schema, Raw: objectWithEmail(t, s, tc.plan)},
			}
			resp := &planmodifier.StringResponse{PlanValue: tc.plan}
			requiresReplaceUnlessAdopting().PlanModifyString(ctx, req, resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if !resp.RequiresReplace {
				t.Error("expected a replacement")
			}
		})
	}
}

// objectWithEmail builds a resource object holding only an email, which is
// all the modifier's checks on the whole object need.
func objectWithEmail(t *testing.T, s resource.SchemaResponse, email types.String) tftypes.Value {
	t.Helper()
	ctx := context.Background()
	objectType := s.Schema.Type().TerraformType(ctx)
	value, err := email.ToTerraformValue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	object, ok := objectType.(tftypes.Object)
	if !ok {
		t.Fatalf("schema type is %T, want an object", objectType)
	}
	attrs := map[string]tftypes.Value{}
	for name, attrType := range object.AttributeTypes {
		attrs[name] = tftypes.NewValue(attrType, nil)
	}
	attrs["email"] = value
	return tftypes.NewValue(objectType, attrs)
}

func jsonUnmarshal(body string, v interface{}) error {
	return json.Unmarshal([]byte(body), v)
}
