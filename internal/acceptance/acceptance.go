package acceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	sdkclient "github.com/appwrite/sdk-for-go/v7/client"
	"github.com/appwrite/sdk-for-go/v7/id"
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

// ExclusiveProjectPreCheck gates tests that change a setting the rest of the
// suite depends on. The acceptance packages run in parallel against one
// project, so turning on a password or sign-up email policy there rejects
// the users other tests are creating at the same moment: example.com counts
// as a disposable domain, and "testpassword123" contains its user's email
// address. These tests stay off unless the project is theirs alone.
func ExclusiveProjectPreCheck(t *testing.T) {
	t.Helper()
	PreCheck(t)
	if os.Getenv("APPWRITE_EXCLUSIVE_PROJECT_TESTS") == "" {
		t.Skip("APPWRITE_EXCLUSIVE_PROJECT_TESTS must be set to run tests that change project-wide settings other tests depend on; point APPWRITE_PROJECT_ID at a project no other test is using")
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

// ServerVersionPreCheck skips a test when the Appwrite server is older than
// minimum, for features a later release added. CI pins an Appwrite version,
// so a test for something newer would otherwise fail there on a 404 for a
// route the server does not have.
func ServerVersionPreCheck(t *testing.T, minimum string) {
	t.Helper()
	PreCheck(t)
	if version, ok := ServerVersionAtLeast(t, minimum); !ok {
		t.Skipf("Appwrite %s is older than %s, which this test needs", version, minimum)
	}
}

// ServerVersionAtLeast reports the Appwrite server's version and whether it is
// minimum or later, for a test of how an older server is handled.
func ServerVersionAtLeast(t *testing.T, minimum string) (string, bool) {
	t.Helper()
	version, err := serverVersion(os.Getenv("APPWRITE_ENDPOINT"))
	if err != nil {
		t.Fatalf("reading the Appwrite server version: %v", err)
	}
	return version, compareVersions(version, minimum) >= 0
}

// serverVersion reads the version /health/version reports. The route takes
// no credentials.
func serverVersion(endpoint string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(endpoint, "/")+"/health/version", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET /health/version returned %s", resp.Status)
	}
	var body struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("decoding /health/version: %w", err)
	}
	if body.Version == "" {
		return "", fmt.Errorf("/health/version reported no version")
	}
	return body.Version, nil
}

// compareVersions compares two dotted numeric versions, such as 2.3.0, and
// returns -1, 0 or 1. A pre-release suffix (2.3.0-rc.1) is ignored, and a
// missing component counts as 0.
func compareVersions(a, b string) int {
	parse := func(v string) []int {
		v, _, _ = strings.Cut(v, "-")
		parts := strings.Split(v, ".")
		out := make([]int, 0, len(parts))
		for _, part := range parts {
			n, _ := strconv.Atoi(part)
			out = append(out, n)
		}
		return out
	}
	pa, pb := parse(a), parse(b)
	for len(pa) < len(pb) {
		pa = append(pa, 0)
	}
	for len(pb) < len(pa) {
		pb = append(pb, 0)
	}
	for i := range pa {
		switch {
		case pa[i] < pb[i]:
			return -1
		case pa[i] > pb[i]:
			return 1
		}
	}
	return 0
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

// ClientWithKey returns an SDK client authenticating with the given key rather
// than the one from the environment. Tests use it to exercise a credential a
// resource just produced, which is the only way to show the credential works.
func ClientWithKey(t *testing.T, apiKey string) sdkclient.Client {
	t.Helper()
	return appwrite.NewClient(
		appwrite.WithEndpoint(os.Getenv("APPWRITE_ENDPOINT")),
		appwrite.WithKey(apiKey),
		appwrite.WithProject(os.Getenv("APPWRITE_PROJECT_ID")),
	)
}

// ClientWithJWT returns an SDK client authenticating as the user the JWT was
// minted for, so a test can show the token actually works rather than that it
// merely looks like a token.
func ClientWithJWT(t *testing.T, jwt string) sdkclient.Client {
	t.Helper()
	return appwrite.NewClient(
		appwrite.WithEndpoint(os.Getenv("APPWRITE_ENDPOINT")),
		appwrite.WithProject(os.Getenv("APPWRITE_PROJECT_ID")),
		appwrite.WithJWT(jwt),
	)
}

// MessagingTarget provisions a user whose email address gives it a messaging
// target, and returns that target's ID along with a cleanup function.
//
// Subscribers subscribe a target to a topic. appwrite_auth_user_target creates
// targets of its own, but the one Appwrite creates implicitly for a user's
// email address is surfaced nowhere -- not on appwrite_auth_user, not on its
// data source -- and cannot be imported, so there is no way to reach it from
// configuration. Creating it through the client keeps subscribing such a
// target under test.
func MessagingTarget(t *testing.T) (targetID string, cleanup func()) {
	t.Helper()

	// Fixtures run before resource.Test, so they sit outside the TF_ACC gate it
	// applies. Without this check a plain `go test ./...` would fail on missing
	// credentials rather than skipping, and with credentials present it would
	// create a live user for a test that is not going to run.
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("acceptance tests skipped unless %s is set", resource.EnvTfAcc)
	}
	PreCheck(t)

	users := appwrite.NewUsers(ProjectClient(t))
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

// ClientForProjectID returns an SDK client scoped to a specific project rather
// than the one in the environment. Destroy checks need it: a test may create
// its own project, and its resources have to be looked up inside that project.
func ClientForProjectID(t *testing.T, projectID string) sdkclient.Client {
	t.Helper()
	return appwrite.NewClient(
		appwrite.WithEndpoint(os.Getenv("APPWRITE_ENDPOINT")),
		appwrite.WithKey(os.Getenv("APPWRITE_API_KEY")),
		appwrite.WithProject(projectID),
	)
}
