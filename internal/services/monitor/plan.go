package monitor

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Type-specific arguments default only for the monitor types that use them
// and are null for every other type, which a schema Default cannot express.
// These modifiers fill in the planned value when the configuration leaves it
// out, so the plan shows what the server will store and removing an argument
// plans it back to its default.

// configuredType returns the monitor type from configuration, and false when
// it is not known yet.
func configuredType(ctx context.Context, config tfsdk.Config) (string, bool) {
	var monitorType types.String
	config.GetAttribute(ctx, path.Root("type"), &monitorType)
	if monitorType.IsUnknown() {
		return "", false
	}
	return monitorType.ValueString(), true
}

type typeDefaultString struct {
	monitorTypes []string
	value        string
}

func defaultStringFor(value string, monitorTypes ...string) planmodifier.String {
	return typeDefaultString{monitorTypes: monitorTypes, value: value}
}

func (m typeDefaultString) Description(context.Context) string {
	return fmt.Sprintf("Defaults to %q for %s monitors and is null for other types.", m.value, strings.Join(m.monitorTypes, ", "))
}

func (m typeDefaultString) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m typeDefaultString) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	monitorType, known := configuredType(ctx, req.Config)
	switch {
	case !known:
		resp.PlanValue = types.StringUnknown()
	case slices.Contains(m.monitorTypes, monitorType):
		resp.PlanValue = types.StringValue(m.value)
	default:
		resp.PlanValue = types.StringNull()
	}
}

type typeDefaultInt64 struct {
	monitorTypes []string
	value        int64
}

func defaultInt64For(value int64, monitorTypes ...string) planmodifier.Int64 {
	return typeDefaultInt64{monitorTypes: monitorTypes, value: value}
}

func (m typeDefaultInt64) Description(context.Context) string {
	return fmt.Sprintf("Defaults to %d for %s monitors and is null for other types.", m.value, strings.Join(m.monitorTypes, ", "))
}

func (m typeDefaultInt64) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m typeDefaultInt64) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if !req.ConfigValue.IsNull() {
		return
	}
	monitorType, known := configuredType(ctx, req.Config)
	switch {
	case !known:
		resp.PlanValue = types.Int64Unknown()
	case slices.Contains(m.monitorTypes, monitorType):
		resp.PlanValue = types.Int64Value(m.value)
	default:
		resp.PlanValue = types.Int64Null()
	}
}

// intervalDefault plans interval when the configuration leaves it out. The
// server picks a probe monitor's interval from the organization's plan, so
// there is no fixed default to plan: a new probe monitor reads it back, an
// existing one keeps the interval it has, and a usage monitor has none.
type intervalDefault struct{}

func (intervalDefault) Description(context.Context) string {
	return "Set by the server for probe monitors when not configured, and null for usage monitors."
}

func (m intervalDefault) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (intervalDefault) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if !req.ConfigValue.IsNull() {
		return
	}
	monitorType, known := configuredType(ctx, req.Config)
	switch {
	case !known:
		resp.PlanValue = types.Int64Unknown()
	case !slices.Contains(probeTypes, monitorType):
		resp.PlanValue = types.Int64Null()
	case !req.StateValue.IsNull():
		resp.PlanValue = req.StateValue
	default:
		resp.PlanValue = types.Int64Unknown()
	}
}
