package dedicateddatabase

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestParamSetters(t *testing.T) {
	p := map[string]interface{}{}
	setStr(p, "a", types.StringValue("x"))
	setStr(p, "skip_null", types.StringNull())
	setStr(p, "skip_unknown", types.StringUnknown())
	setInt(p, "n", types.Int64Value(3))
	setInt(p, "skip_int", types.Int64Null())
	setBool(p, "b", types.BoolValue(true))
	setBool(p, "skip_bool", types.BoolNull())

	if len(p) != 3 {
		t.Fatalf("expected 3 params, got %d: %v", len(p), p)
	}
	if p["a"] != "x" || p["n"] != 3 || p["b"] != true {
		t.Fatalf("wrong values: %v", p)
	}
}

func TestSplitTwo(t *testing.T) {
	cases := []struct {
		in    string
		a, b  string
		valid bool
	}{
		{"postgresql/abc", "postgresql", "abc", true},
		{"mysql/db/policy", "mysql", "db/policy", true}, // only first slash splits
		{"noslash", "", "", false},
		{"/trailing", "", "", false},
		{"leading/", "", "", false},
	}
	for _, c := range cases {
		a, b, ok := splitTwo(c.in)
		if ok != c.valid || a != c.a || b != c.b {
			t.Errorf("splitTwo(%q) = (%q,%q,%v), want (%q,%q,%v)", c.in, a, b, ok, c.a, c.b, c.valid)
		}
	}
}

func TestValidEnginesDeterministic(t *testing.T) {
	if got := validEngines(); got != "mongo, mysql, postgresql" {
		t.Fatalf("validEngines() = %q, want sorted list", got)
	}
}
