package column

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// useStateForUnknownUnlessUpdating copies the prior-state value into the plan
// when the planned value is unknown, EXCEPT when this is an update where
// something else in the resource is actually changing — in which case the
// value must be left unknown so the API's new timestamp doesn't violate the
// plan→apply contract.
func useStateForUnknownUnlessUpdating() planmodifier.String {
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
	// If prior state and plan differ on any attribute, this is an actual update —
	// leave the timestamp unknown so "(known after apply)" appears in the plan
	// and the API's new value won't cause an inconsistent-result error.
	if !req.State.Raw.Equal(req.Plan.Raw) {
		return
	}
	// No-op refresh: reuse the prior state value to keep plans clean.
	resp.PlanValue = types.StringValue(req.StateValue.ValueString())
}
