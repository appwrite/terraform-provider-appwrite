package common

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// CreateContext derives a context bounded by the resource's create timeout.
//
// The three helpers here exist so a resource does not repeat the
// read-the-timeout, handle-the-diagnostics, defer-the-cancel dance in every
// method -- and, more importantly, so that forgetting the cancel is not possible
// per call site. The returned cancel must still be deferred; that is the one
// part a helper cannot take away.
func CreateContext(ctx context.Context, t timeouts.Value, fallback time.Duration) (context.Context, context.CancelFunc, diag.Diagnostics) {
	d, diags := t.Create(ctx, fallback)
	if diags.HasError() {
		return ctx, func() {}, diags
	}
	derived, cancel := context.WithTimeout(ctx, d)
	return derived, cancel, diags
}

// UpdateContext derives a context bounded by the resource's update timeout.
func UpdateContext(ctx context.Context, t timeouts.Value, fallback time.Duration) (context.Context, context.CancelFunc, diag.Diagnostics) {
	d, diags := t.Update(ctx, fallback)
	if diags.HasError() {
		return ctx, func() {}, diags
	}
	derived, cancel := context.WithTimeout(ctx, d)
	return derived, cancel, diags
}

// DeleteContext derives a context bounded by the resource's delete timeout.
func DeleteContext(ctx context.Context, t timeouts.Value, fallback time.Duration) (context.Context, context.CancelFunc, diag.Diagnostics) {
	d, diags := t.Delete(ctx, fallback)
	if diags.HasError() {
		return ctx, func() {}, diags
	}
	derived, cancel := context.WithTimeout(ctx, d)
	return derived, cancel, diags
}
