package common

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
)

// sentHeaders issues one request through an SDK client configured with
// WithIdentity and returns the headers Appwrite would have received.
func sentHeaders(t *testing.T, providerVersion string, terraformVersion string) http.Header {
	t.Helper()

	var received http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	clt := appwrite.NewClient(
		appwrite.WithEndpoint(server.URL),
		WithIdentity(providerVersion, terraformVersion),
	)
	if _, err := clt.Call(http.MethodGet, "/health", nil, nil); err != nil {
		t.Fatalf("request: %v", err)
	}
	if received == nil {
		t.Fatal("the server received no request")
	}
	return received
}

// Requests used to report x-sdk-name "Go", so Appwrite attributed provider
// traffic to the Go SDK and only the User-Agent told them apart.
func TestWithIdentityReportsTerraform(t *testing.T) {
	t.Setenv("TF_APPEND_USER_AGENT", "")

	headers := sentHeaders(t, "2.1.0", "1.9.5")

	if got := headers.Get("X-SDK-Name"); got != "Terraform" {
		t.Errorf("x-sdk-name = %q, want Terraform", got)
	}
	if got := headers.Get("X-SDK-Language"); got != "terraform" {
		t.Errorf("x-sdk-language = %q, want terraform", got)
	}
	if got := headers.Get("X-SDK-Platform"); got != "server" {
		t.Errorf("x-sdk-platform = %q, want server", got)
	}
	if got := headers.Get("X-SDK-Version"); got != "2.1.0" {
		t.Errorf("x-sdk-version = %q, want the provider version", got)
	}
}

func TestWithIdentityUserAgent(t *testing.T) {
	t.Setenv("TF_APPEND_USER_AGENT", "terragrunt/0.55.0")

	userAgent := sentHeaders(t, "2.1.0", "1.9.5").Get("User-Agent")

	if !strings.HasPrefix(userAgent, "Terraform/1.9.5 terraform-provider-appwrite/2.1.0 ") {
		t.Errorf("user-agent = %q, want it to lead with Terraform core and then the provider", userAgent)
	}
	if !strings.Contains(userAgent, "AppwriteGoSDK/") {
		t.Errorf("user-agent = %q, want the underlying SDK kept traceable", userAgent)
	}
	if !strings.HasSuffix(userAgent, " terragrunt/0.55.0") {
		t.Errorf("user-agent = %q, want TF_APPEND_USER_AGENT last", userAgent)
	}
}

// Terraform core does not always report its version; a blank one must not
// leave an empty "Terraform/" token.
func TestWithIdentityUserAgentWithoutTerraformVersion(t *testing.T) {
	t.Setenv("TF_APPEND_USER_AGENT", "")

	userAgent := sentHeaders(t, "2.1.0", "").Get("User-Agent")

	if strings.Contains(userAgent, "Terraform/") {
		t.Errorf("user-agent = %q, want no Terraform token", userAgent)
	}
	if !strings.HasPrefix(userAgent, "terraform-provider-appwrite/2.1.0 ") {
		t.Errorf("user-agent = %q, want it to lead with the provider", userAgent)
	}
}
