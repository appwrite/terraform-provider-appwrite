package acceptance

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/appwrite/terraform-provider-appwrite/internal/provider"
)

var ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"appwrite": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func PreCheck(t *testing.T) {
	t.Helper()
	preCheckBase(t)
	if os.Getenv("APPWRITE_PROJECT_ID") == "" {
		t.Fatal("APPWRITE_PROJECT_ID must be set for acceptance tests")
	}
}

func OrganizationPreCheck(t *testing.T) {
	t.Helper()
	preCheckBase(t)
	if os.Getenv("APPWRITE_ORGANIZATION_ID") == "" {
		t.Skip("APPWRITE_ORGANIZATION_ID must be set for organization acceptance tests")
	}
}

// DedicatedPreCheck gates the dedicated database tests. They provision real,
// billable infrastructure and take minutes per step, so they stay off unless
// explicitly asked for.
func DedicatedPreCheck(t *testing.T) {
	t.Helper()
	PreCheck(t)
	if os.Getenv("APPWRITE_DEDICATED_DATABASE_TESTS") == "" {
		t.Skip("APPWRITE_DEDICATED_DATABASE_TESTS must be set to run dedicated database acceptance tests; they provision billable infrastructure")
	}
}

func preCheckBase(t *testing.T) {
	t.Helper()
	if os.Getenv("APPWRITE_ENDPOINT") == "" {
		t.Fatal("APPWRITE_ENDPOINT must be set for acceptance tests")
	}
	if os.Getenv("APPWRITE_API_KEY") == "" {
		t.Fatal("APPWRITE_API_KEY must be set for acceptance tests")
	}
}
