package common

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// UseStateForUnknownUnlessUpdating copies the prior-state value into the plan
// when the planned value is unknown, EXCEPT when this is an update where
// something else in the resource is actually changing — in which case the
// value must be left unknown so the API's new timestamp doesn't violate the
// plan→apply contract.
func UseStateForUnknownUnlessUpdating() planmodifier.String {
	return useStateForUnknownUnlessUpdatingModifier{}
}

type useStateForUnknownUnlessUpdatingModifier struct{}

func (m useStateForUnknownUnlessUpdatingModifier) Description(_ context.Context) string {
	return "Uses prior state when planned value is unknown, except on updates."
}

func (m useStateForUnknownUnlessUpdatingModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m useStateForUnknownUnlessUpdatingModifier) PlanModifyString(
	_ context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	// Create: state is null, leave unknown so the API fills it in.
	if req.State.Raw.IsNull() {
		return
	}
	// Destroy: nothing to do.
	if req.Plan.Raw.IsNull() {
		return
	}
	// Only act when the planned value is unknown.
	if !req.PlanValue.IsUnknown() {
		return
	}

	// We cannot compare req.State.Raw and req.Plan.Raw directly: computed
	// attributes (including this one) are unknown in the plan, so
	// tftypes.Value.Equal always returns false. Instead, compare only the
	// attributes that have known values in the plan — those are the
	// user-configured attributes. If any of them differ from the prior
	// state, this is a real update.
	var stateMap, planMap map[string]tftypes.Value
	if err := req.State.Raw.As(&stateMap); err != nil {
		return
	}
	if err := req.Plan.Raw.As(&planMap); err != nil {
		return
	}
	for k, pv := range planMap {
		if !pv.IsKnown() {
			continue
		}
		sv, ok := stateMap[k]
		if !ok || !sv.Equal(pv) {
			// A known attribute changed — leave the timestamp unknown so
			// "(known after apply)" appears in the plan.
			return
		}
	}

	// No-op refresh: reuse the prior state value to keep plans clean.
	resp.PlanValue = types.StringValue(req.StateValue.ValueString())
}
