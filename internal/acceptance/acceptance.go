package acceptance

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	sdkclient "github.com/appwrite/sdk-for-go/v7/client"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/appwrite/terraform-provider-appwrite/internal/provider"
)

var ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"appwrite": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// ProtoV6ProviderFactoriesWithEcho adds the echo provider alongside this one.
// Ephemeral resources never reach state, so there is nothing for a state check
// to assert against; echo copies the value it is given into a managed resource
// so the test can inspect it.
var ProtoV6ProviderFactoriesWithEcho = map[string]func() (tfprotov6.ProviderServer, error){
	"appwrite": providerserver.NewProtocol6WithError(provider.New("test")()),
	"echo":     echoprovider.NewProviderServer(),
}

// ProjectClient returns an SDK client for the acceptance test project, built
// from the same environment the provider reads. Tests use it to assert against
// the server directly, which is the only way to check the effect of an
// ephemeral resource's Close: by then nothing remains in state to inspect.
func ProjectClient(t *testing.T) sdkclient.Client {
	t.Helper()
	return appwrite.NewClient(
		appwrite.WithEndpoint(os.Getenv("APPWRITE_ENDPOINT")),
		appwrite.WithKey(os.Getenv("APPWRITE_API_KEY")),
		appwrite.WithProject(os.Getenv("APPWRITE_PROJECT_ID")),
	)
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

// User provisions an Appwrite user directly through the client and returns its
// ID with a cleanup function.
//
// Fixtures exist so that a test can assert about something Terraform did not
// itself destroy. Checking that an ephemeral session was closed, for instance,
// is meaningless if Terraform deleted the user and took its sessions along with
// it: the session would be gone either way.
func User(t *testing.T) (userID string, cleanup func()) {
	t.Helper()

	// Fixtures run before resource.Test, so they sit outside the TF_ACC gate it
	// applies. Without this a plain `go test ./...` would fail on missing
	// credentials rather than skipping, and with credentials present it would
	// create a live user for a test that is not going to run.
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("acceptance tests skipped unless %s is set", resource.EnvTfAcc)
	}
	PreCheck(t)

	users := appwrite.NewUsers(ProjectClient(t))
	id := fmt.Sprintf("fixture-%d", time.Now().UnixNano())

	user, err := users.Create(id, users.WithCreateEmail(fmt.Sprintf("%s@example.com", id)))
	if err != nil {
		t.Fatalf("creating fixture user: %v", err)
	}
	return user.Id, func() {
		if _, err := users.Delete(user.Id); err != nil {
			t.Logf("could not delete fixture user %s: %v", user.Id, err)
		}
	}
}
