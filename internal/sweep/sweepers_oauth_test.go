package sweep

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
)

// TestSweepOAuthProviders checks that only providers holding a test's prefix
// are reset, and that each reset clears what that provider holds.
func TestSweepOAuthProviders(t *testing.T) {
	var mu sync.Mutex
	patched := map[string]map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"total":4,"providers":[
				{"$id":"github","enabled":true,"clientId":"Iv1.tf-acc-test","clientSecret":""},
				{"$id":"amazon","enabled":true,"clientId":"production-app","clientSecret":""},
				{"$id":"apple","enabled":false,"serviceId":"","keyId":"","teamId":"","p8File":"","nativeEnabled":true,"nativeClientIds":["com.example.tf-acc-test"]},
				{"$id":"oidc","enabled":true,"clientId":"tf-acc-test","clientSecret":"","wellKnownURL":"https://idp.example.com","prompt":["login"],"maxAge":60}
			]}`))
			return
		}
		body := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		patched[r.URL.Path] = body
		mu.Unlock()
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	t.Setenv("APPWRITE_ENDPOINT", server.URL)
	t.Setenv("APPWRITE_API_KEY", "standard_test")
	t.Setenv("APPWRITE_SWEEP_PROJECT_ID", "test-project")

	if err := sweepOAuthProviders(""); err != nil {
		t.Fatal(err)
	}

	want := map[string]map[string]any{
		"/project/oauth2/github": {"enabled": false, "clientId": "", "clientSecret": ""},
		"/project/oauth2/apple": {
			"enabled": false, "serviceId": "", "keyId": "", "teamId": "", "p8File": "",
			"nativeEnabled": false, "nativeClientIds": []any{},
		},
		"/project/oauth2/oidc": {"enabled": false, "clientId": "", "clientSecret": "", "wellKnownURL": ""},
	}
	if !reflect.DeepEqual(patched, want) {
		t.Errorf("reset requests:\n got %v\nwant %v", patched, want)
	}
}
