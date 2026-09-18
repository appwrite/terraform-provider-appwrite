package sweep_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/appwrite/terraform-provider-appwrite/internal/sweep"
)

// TestMain gives the sweepers a home to run from. resource.TestMain handles the
// -sweep flags and exits without running any test when sweeping, so an ordinary
// `go test ./...` reaches TestSweepablePrefix below and nothing else.
func TestMain(m *testing.M) {
	sweep.Register()
	resource.TestMain(m)
}
