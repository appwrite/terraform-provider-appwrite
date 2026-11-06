package project

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/client"
)

type capturedRequest struct {
	method  string
	path    string
	project string
	body    map[string]any
}

// oauthServer answers every request with a provider and keeps the requests it
// received.
func oauthServer(t *testing.T) (*httptest.Server, func() []capturedRequest) {
	t.Helper()

	var mu sync.Mutex
	var requests []capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		requests = append(requests, capturedRequest{
			method:  r.Method,
			path:    r.URL.Path,
			project: r.Header.Get("X-Appwrite-Project"),
			body:    body,
		})
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"$id": strings.TrimPrefix(r.URL.Path, "/project/oauth2/"), "enabled": true})
	}))
	t.Cleanup(server.Close)
	return server, func() []capturedRequest {
		mu.Lock()
		defer mu.Unlock()
		return append([]capturedRequest(nil), requests...)
	}
}

func testOAuthClient(endpoint string) client.Client {
	return appwrite.NewClient(appwrite.WithEndpoint(endpoint), appwrite.WithProject("test-project"), appwrite.WithKey("test-key"))
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestOAuthProvidersMatchSDK checks the provider table against the SDK.
//
// The SDK has one UpdateOAuth2<Provider> method per provider. For each, it is
// called with every option it has, and the route and parameter names it sends
// must be the ones the table sends for a configuration setting every argument
// the provider takes. A provider the SDK gains on an upgrade, or a parameter
// it renames, fails here rather than against a server.
func TestOAuthProvidersMatchSDK(t *testing.T) {
	server, requests := oauthServer(t)
	service := reflect.ValueOf(appwrite.NewProject(testOAuthClient(server.URL)))
	serviceType := service.Type()

	sdkLabels := map[string]bool{}
	for i := 0; i < serviceType.NumMethod(); i++ {
		label, ok := strings.CutPrefix(serviceType.Method(i).Name, "UpdateOAuth2")
		// UpdateOAuth2Server configures Appwrite as an OAuth2 server, not a
		// sign-in provider.
		if ok && label != "Server" {
			sdkLabels[label] = true
		}
	}
	tableLabels := map[string]bool{}
	for _, p := range oauthProviders {
		tableLabels[p.label] = true
	}
	for label := range sdkLabels {
		if !tableLabels[label] {
			t.Errorf("the SDK has UpdateOAuth2%s, which oauthProviders does not list", label)
		}
	}
	for label := range tableLabels {
		if !sdkLabels[label] {
			t.Errorf("oauthProviders lists %s, which the SDK has no UpdateOAuth2%s for", label, label)
		}
	}

	for _, p := range oauthProviders {
		method := service.MethodByName("UpdateOAuth2" + p.label)
		if !method.IsValid() {
			continue
		}
		optionType := method.Type().In(0).Elem()

		// Every setter of this method's option type, whatever its name.
		// Matching on the name alone would also pick up PaypalSandbox's
		// setters for Paypal.
		var options []reflect.Value
		for i := 0; i < serviceType.NumMethod(); i++ {
			setter := service.Method(i)
			if !strings.HasPrefix(serviceType.Method(i).Name, "WithUpdateOAuth2") || setter.Type().NumOut() != 1 || setter.Type().Out(0) != optionType {
				continue
			}
			options = append(options, setter.Call([]reflect.Value{sampleArgument(t, setter.Type().In(0))})[0])
		}

		before := len(requests())
		out := method.Call(options)
		if err, _ := out[1].Interface().(error); err != nil {
			t.Fatalf("UpdateOAuth2%s: %v", p.label, err)
		}
		sent := requests()[before]

		plan := fullModel(p)
		want := updateParams(p, &plan, "secret", true)
		if sent.method != http.MethodPatch || sent.path != oauthProviderPath(p.id) {
			t.Errorf("%s: SDK sends %s %s, table sends PATCH %s", p.id, sent.method, sent.path, oauthProviderPath(p.id))
		}
		if got, want := keys(sent.body), keys(want); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: SDK sends parameters %v, table sends %v", p.id, got, want)
		}
	}
}

func sampleArgument(t *testing.T, typ reflect.Type) reflect.Value {
	t.Helper()
	switch typ.Kind() {
	case reflect.String:
		return reflect.ValueOf("value")
	case reflect.Bool:
		return reflect.ValueOf(true)
	case reflect.Int:
		return reflect.ValueOf(1)
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.String {
			return reflect.ValueOf([]string{"value"})
		}
	}
	t.Fatalf("no sample value for an SDK option of type %s", typ)
	return reflect.Value{}
}

// TestOAuthProviderRoutes sends every provider's update with every argument
// set, then its reset, and checks the route and the values that reach the
// server.
func TestOAuthProviderRoutes(t *testing.T) {
	t.Parallel()

	for _, p := range oauthProviders {
		t.Run(p.id, func(t *testing.T) {
			t.Parallel()
			server, requests := oauthServer(t)
			c := testOAuthClient(server.URL)

			plan := fullModel(p)
			if _, err := updateOAuthProvider(c, p.id, updateParams(p, &plan, "secret-"+p.id, true)); err != nil {
				t.Fatalf("update: %v", err)
			}
			if _, err := updateOAuthProvider(c, p.id, resetParams(p)); err != nil {
				t.Fatalf("reset: %v", err)
			}
			if _, err := getOAuthProvider(c, p.id); err != nil {
				t.Fatalf("get: %v", err)
			}

			sent := requests()
			if len(sent) != 3 {
				t.Fatalf("got %d requests, want 3", len(sent))
			}
			route := "/project/oauth2/" + p.id
			for i, wantMethod := range []string{http.MethodPatch, http.MethodPatch, http.MethodGet} {
				if sent[i].method != wantMethod || sent[i].path != route || sent[i].project != "test-project" {
					t.Errorf("request %d: got %s %s for project %q, want %s %s for test-project",
						i, sent[i].method, sent[i].path, sent[i].project, wantMethod, route)
				}
			}

			update := map[string]any{
				"enabled":         true,
				p.clientIDParam(): "client-" + p.id,
				p.secretParam():   "secret-" + p.id,
			}
			reset := map[string]any{
				"enabled":         false,
				p.clientIDParam(): "",
				p.secretParam():   "",
			}
			for _, name := range p.extras {
				extra := oauthExtras[name]
				switch extra.kind {
				case extraString:
					update[extra.key] = name + "-value"
					reset[extra.key] = ""
				case extraBool:
					update[extra.key] = true
					reset[extra.key] = false
				case extraInt64:
					update[extra.key] = float64(3600)
				case extraSet:
					update[extra.key] = []any{name + "-a", name + "-b"}
					if !extra.serverDefault {
						reset[extra.key] = []any{}
					}
				}
			}
			if !reflect.DeepEqual(sent[0].body, update) {
				t.Errorf("update body:\n got %v\nwant %v", sent[0].body, update)
			}
			if !reflect.DeepEqual(sent[1].body, reset) {
				t.Errorf("reset body:\n got %v\nwant %v", sent[1].body, reset)
			}
		})
	}
}

// TestOAuthProviderResetBodies pins the reset of the providers with the most
// to clear, written out rather than derived from the table.
func TestOAuthProviderResetBodies(t *testing.T) {
	t.Parallel()

	for id, want := range map[string]map[string]any{
		"github":  {"enabled": false, "clientId": "", "clientSecret": ""},
		"dropbox": {"enabled": false, "appKey": "", "appSecret": ""},
		// prompt and maxAge cannot be cleared, so they are left out.
		"oidc": {
			"enabled": false, "clientId": "", "clientSecret": "",
			"wellKnownURL": "", "authorizationURL": "", "tokenURL": "", "userInfoURL": "",
		},
		"apple": {
			"enabled": false, "serviceId": "", "p8File": "", "keyId": "", "teamId": "",
			"nativeClientIds": []string{}, "nativeEnabled": false,
		},
		"google": {
			"enabled": false, "clientId": "", "clientSecret": "",
			"nativeClientIds": []string{}, "nativeEnabled": false,
		},
		"keycloak": {"enabled": false, "clientId": "", "clientSecret": "", "endpoint": "", "realmName": ""},
		"okta":     {"enabled": false, "clientId": "", "clientSecret": "", "domain": "", "authorizationServerId": ""},
	} {
		p, _ := lookupOAuthProvider(id)
		if got := resetParams(p); !reflect.DeepEqual(got, want) {
			t.Errorf("%s reset:\n got %v\nwant %v", id, got, want)
		}
	}
}

// TestOAuthProviderUnsetArguments checks what a configuration naming only the
// provider sends: every argument Appwrite can clear, cleared, so removing one
// from the configuration removes it on the server; no secret, so one set
// outside Terraform survives; and none of the settings Appwrite cannot clear.
func TestOAuthProviderUnsetArguments(t *testing.T) {
	t.Parallel()

	for id, want := range map[string]map[string]any{
		"github": {"enabled": true, "clientId": ""},
		"oidc": {
			"enabled": true, "clientId": "",
			"wellKnownURL": "", "authorizationURL": "", "tokenURL": "", "userInfoURL": "",
		},
		"google": {"enabled": true, "clientId": "", "nativeClientIds": []string{}, "nativeEnabled": false},
	} {
		p, _ := lookupOAuthProvider(id)
		plan := emptyModel(p)
		if got := updateParams(p, &plan, "", false); !reflect.DeepEqual(got, want) {
			t.Errorf("%s:\n got %v\nwant %v", id, got, want)
		}
	}
}
