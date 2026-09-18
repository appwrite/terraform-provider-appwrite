package acceptance

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// ResourceTest runs an acceptance test case with the guards every resource test
// should have, whether or not its author remembered them.
//
// Three things are injected. A CheckDestroy that asks the server whether each
// resource in state actually went away, because the framework only checks that
// Terraform removed it from state -- a Delete that returns success without
// deleting anything passes an unguarded test. A plan check on every
// configuration step asserting nothing is being replaced, so a resource cannot
// quietly start recreating itself. And refresh-after-apply, which catches the
// common framework bug where Create writes a value to state that Read does not
// produce, and which otherwise only surfaces as a perpetual diff for users.
//
// Tests call this rather than resource.Test directly; scripts/checks/no-direct-resource-test.sh
// enforces that, because a guard that can be skipped by writing the obvious
// thing instead is not a guard.
func ResourceTest(t *testing.T, tc resource.TestCase) {
	t.Helper()
	runResourceTest(t, tc, true)
}

// ResourceTestAllowingReplace is ResourceTest without the no-replace plan
// check, for the tests whose subject is replacement itself -- changing a
// ForceNew argument and asserting the resource is recreated. It keeps the
// destroy check and refresh-after-apply.
func ResourceTestAllowingReplace(t *testing.T, tc resource.TestCase) {
	t.Helper()
	runResourceTest(t, tc, false)
}

func runResourceTest(t *testing.T, tc resource.TestCase, guardReplace bool) {
	t.Helper()

	// Terraform reads state back after every apply and fails the step if the
	// refreshed state differs. Set here rather than in CI so a local run and a
	// nightly run agree about what passing means.
	t.Setenv(resource.EnvTfAccRefreshAfterApply, "1")

	if tc.CheckDestroy == nil {
		tc.CheckDestroy = CheckDestroy(t)
	}

	if guardReplace {
		for i := range tc.Steps {
			// Import and destroy steps have no configuration to plan, and a
			// taint step legitimately replaces. Only guard steps that apply a
			// configuration.
			if tc.Steps[i].Config == "" && tc.Steps[i].ConfigFile == nil && tc.Steps[i].ConfigDirectory == nil {
				continue
			}
			if tc.Steps[i].Taint != nil {
				continue
			}
			tc.Steps[i].ConfigPlanChecks.PreApply = append(
				tc.Steps[i].ConfigPlanChecks.PreApply,
				NoUnexpectedReplace(),
			)
		}
	}

	resource.Test(t, tc)
}

// ExpectEmptyPlanAfterApply is the step to append to a test that wants to prove
// its configuration is convergent beyond what refresh-after-apply already
// checks. Refresh catches Read disagreeing with Create; this catches the plan
// itself never settling.
func ExpectEmptyPlanAfterApply(config string) resource.TestStep {
	return resource.TestStep{
		Config: config,
		ConfigPlanChecks: resource.ConfigPlanChecks{
			PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
		},
	}
}

// acceptanceEnabled reports whether the suite is running for real. Fixtures and
// destroy checks reach the network, so they have to agree with the gate
// resource.Test applies internally.
func acceptanceEnabled() bool {
	return os.Getenv(resource.EnvTfAcc) != ""
}
