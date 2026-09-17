package acceptance

import (
	"fmt"
	"os"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	sdkclient "github.com/appwrite/sdk-for-go/v7/client"
	"github.com/appwrite/sdk-for-go/v7/id"
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

// CloudPreCheck gates tests for features that only Appwrite Cloud serves. The
// backups API, for one, is not routed at all on a self-hosted install, so the
// requests come back as 404 HTML rather than an API error. CI runs against
// self-hosted Appwrite, so these stay off unless explicitly asked for.
func CloudPreCheck(t *testing.T) {
	t.Helper()
	PreCheck(t)
	if os.Getenv("APPWRITE_CLOUD_TESTS") == "" {
		t.Skip("APPWRITE_CLOUD_TESTS must be set to run tests for Cloud-only features")
	}
}

// BuildPreCheck gates tests that make Appwrite build a deployment. Those need
// the orchestrator and runtime executor running alongside the API, and they
// fetch a template repository over the network and build it, which takes
// minutes and fails whenever the upstream repository or registry is having a
// bad day. Too slow and too flaky to sit in the nightly.
func BuildPreCheck(t *testing.T) {
	t.Helper()
	PreCheck(t)
	if os.Getenv("APPWRITE_BUILD_TESTS") == "" {
		t.Skip("APPWRITE_BUILD_TESTS must be set to run deployment build tests; they require the build stack and take minutes")
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

// client builds an Appwrite SDK client from the same environment the provider
// reads. Acceptance fixtures use it to provision prerequisites that no provider
// resource can create.
func newClient(t *testing.T) sdkclient.Client {
	t.Helper()
	return appwrite.NewClient(
		appwrite.WithEndpoint(os.Getenv("APPWRITE_ENDPOINT")),
		appwrite.WithKey(os.Getenv("APPWRITE_API_KEY")),
		appwrite.WithProject(os.Getenv("APPWRITE_PROJECT_ID")),
	)
}

// MessagingTarget provisions a user whose email address gives it a messaging
// target, and returns that target's ID along with a cleanup function.
//
// Subscribers subscribe a target to a topic, but targets are created implicitly
// alongside a user and the provider surfaces them nowhere -- not on
// appwrite_auth_user, not on its data source -- so there is no way to reach one
// from configuration. Creating it through the client keeps the subscriber
// resource under test rather than skipping its coverage.
func MessagingTarget(t *testing.T) (targetID string, cleanup func()) {
	t.Helper()
	PreCheck(t)

	users := appwrite.NewUsers(newClient(t))
	userID := id.Unique()

	user, err := users.Create(userID, users.WithCreateEmail(fmt.Sprintf("%s@example.com", userID)))
	if err != nil {
		t.Fatalf("creating fixture user for messaging target: %v", err)
	}
	cleanup = func() {
		if _, err := users.Delete(user.Id); err != nil {
			t.Logf("could not delete fixture user %s: %v", user.Id, err)
		}
	}

	for _, target := range user.Targets {
		if target.ProviderType == "email" {
			return target.Id, cleanup
		}
	}

	cleanup()
	t.Fatalf("user %s was created without an email target", user.Id)
	return "", func() {}
}
