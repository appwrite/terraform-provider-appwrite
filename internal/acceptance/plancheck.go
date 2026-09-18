package acceptance

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

var _ plancheck.PlanCheck = noUnexpectedReplace{}

type noUnexpectedReplace struct{}

// CheckPlan fails when the plan would destroy and recreate any resource.
//
// A resource that silently starts replacing itself is the most expensive bug
// this provider can ship: the plan still applies cleanly, the test still
// passes, and the first person to find out is a user whose database was
// recreated. The check runs on every step of every test rather than being
// opted into, because the bug arrives by accident.
func (noUnexpectedReplace) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	var result []error

	for _, rc := range req.Plan.ResourceChanges {
		if rc.Change == nil {
			continue
		}
		if rc.Change.Actions.Replace() {
			result = append(result, fmt.Errorf(
				"%s is planned for replacement (%v); if this is intended, use acceptance.ResourceTestAllowingReplace",
				rc.Address, rc.Change.Actions,
			))
		}
	}

	resp.Error = errors.Join(result...)
}

// NoUnexpectedReplace returns a plan check asserting that nothing in the plan
// is being destroyed and recreated.
func NoUnexpectedReplace() plancheck.PlanCheck {
	return noUnexpectedReplace{}
}
