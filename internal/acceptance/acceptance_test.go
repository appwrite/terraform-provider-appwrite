package acceptance

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"2.2.0", "2.3.0", -1},
		{"2.3.0", "2.3.0", 0},
		{"2.7.0", "2.3.0", 1},
		{"2.10.0", "2.9.0", 1},
		{"3.0", "2.99.99", 1},
		{"2.3", "2.3.0", 0},
		{"2.3.0-rc.1", "2.3.0", 0},
		{"1.9.10", "2.3.0", -1},
	} {
		if got := compareVersions(tc.a, tc.b); got != tc.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestServerVersion(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/health/version" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"2.2.0"}`))
	}))
	t.Cleanup(server.Close)

	for _, endpoint := range []string{server.URL + "/v1", server.URL + "/v1/"} {
		got, err := serverVersion(endpoint)
		if err != nil || got != "2.2.0" {
			t.Errorf("serverVersion(%q) = %q, %v; want 2.2.0", endpoint, got, err)
		}
	}
	if _, err := serverVersion(server.URL); err == nil {
		t.Error("serverVersion succeeded against an endpoint without /v1")
	}
}
