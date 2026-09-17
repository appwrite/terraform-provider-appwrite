package common

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// WriteOnlyValue reads a write-only attribute from the configuration.
//
// A write-only argument is deliberately absent from both plan and state, so it
// cannot be read the usual way; the configuration is the only place it exists,
// and only during apply. Returns a null value when the attribute is unset.
func WriteOnlyValue(ctx context.Context, config tfsdk.Config, attribute string, diagnostics *diag.Diagnostics) types.String {
	var value types.String
	diagnostics.Append(config.GetAttribute(ctx, path.Root(attribute), &value)...)
	return value
}

// ResolveSecret picks between a write-only argument and its state-stored
// counterpart, preferring the write-only one. Schema-level validation is what
// stops both being set at once; this only decides which to use.
func ResolveSecret(writeOnly types.String, stored types.String) string {
	if !writeOnly.IsNull() && !writeOnly.IsUnknown() {
		return writeOnly.ValueString()
	}
	return stored.ValueString()
}
