package common_test

import (
	"context"
	"strings"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestVariableKeyValidators(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{name: "uppercase", key: "API_URL", valid: true},
		{name: "leading underscore", key: "_PRIVATE", valid: true},
		{name: "trailing digit", key: "KEY1", valid: true},
		{name: "max length", key: strings.Repeat("A", 255), valid: true},
		{name: "hyphen", key: "MY-KEY", valid: false},
		{name: "dot", key: "MY.KEY", valid: false},
		{name: "space", key: "MY KEY", valid: false},
		{name: "leading digit", key: "9KEY", valid: false},
		{name: "accent", key: "KÉY", valid: false},
		{name: "tab", key: "KEY\t", valid: false},
		{name: "empty", key: "", valid: false},
		{name: "too long", key: strings.Repeat("A", 256), valid: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("key"),
				ConfigValue: types.StringValue(tc.key),
			}
			resp := &validator.StringResponse{}

			for _, v := range common.VariableKeyValidators() {
				v.ValidateString(context.Background(), req, resp)
			}

			if got := !resp.Diagnostics.HasError(); got != tc.valid {
				t.Errorf("key %q: got valid=%v, want valid=%v (%s)", tc.key, got, tc.valid, resp.Diagnostics)
			}
		})
	}
}
