package provider

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/client"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type capturedRequest struct {
	method  string
	path    string
	project string
	body    map[string]any
}

// providerServer answers every request with a provider of the given service
// and keeps the requests it received.
func providerServer(t *testing.T, service string) (*httptest.Server, *[]capturedRequest) {
	t.Helper()

	var requests []capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		requests = append(requests, capturedRequest{
			method:  r.Method,
			path:    r.URL.Path,
			project: r.Header.Get("X-Appwrite-Project"),
			body:    body,
		})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"$id":      "prov-id",
			"name":     "name",
			"provider": service,
			"enabled":  true,
			"options":  map[string]any{},
		})
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

func testClient(url string) client.Client {
	// No retries: the SDK client here is bare, so each request reaches the
	// server exactly once.
	return appwrite.NewClient(appwrite.WithEndpoint(url+"/v1"), appwrite.WithProject("project-id"))
}

// TestProviderRoutes sends a create and an update for every provider type and
// checks the route and the body that reach the server.
//
// Each type has its own route, its own SDK call and its own argument names,
// so a type wired to the wrong call or an argument passed under the wrong
// name would otherwise only show up against a real server. The acceptance
// tests reach only some types, and the appwrite type not at all on the
// Appwrite version CI runs, so this is the coverage every type gets in CI.
func TestProviderRoutes(t *testing.T) {
	t.Parallel()

	s := types.StringValue
	b := types.BoolValue
	i := types.Int64Value

	email := func(plan *providerResourceModel) {
		plan.FromEmail = s("from@example.com")
		plan.FromName = s("From")
		plan.ReplyToEmail = s("reply@example.com")
		plan.ReplyToName = s("Reply")
	}
	emailBody := map[string]any{
		"fromEmail":    "from@example.com",
		"fromName":     "From",
		"replyToEmail": "reply@example.com",
		"replyToName":  "Reply",
	}

	for _, tc := range []struct {
		providerType string
		configure    func(*providerResourceModel)
		// body lists the type-specific arguments both routes must receive,
		// beyond the name and enabled every route takes.
		body map[string]any
		// updated, when set, changes the plan before the update, and
		// updateBody then replaces body for the update request.
		updated    func(*providerResourceModel)
		updateBody map[string]any
	}{
		{
			providerType: "sendgrid",
			configure:    func(p *providerResourceModel) { p.APIKey = s("key"); email(p) },
			body:         merge(map[string]any{"apiKey": "key"}, emailBody),
		},
		{
			providerType: "mailgun",
			configure: func(p *providerResourceModel) {
				p.APIKey = s("key")
				p.Domain = s("mg.example.com")
				p.IsEuRegion = b(true)
				email(p)
			},
			body: merge(map[string]any{"apiKey": "key", "domain": "mg.example.com", "isEuRegion": true}, emailBody),
		},
		{
			providerType: "smtp",
			configure: func(p *providerResourceModel) {
				p.Host = s("smtp.example.com")
				p.Port = i(587)
				p.Username = s("user")
				p.Password = s("pass")
				p.Encryption = s("tls")
				p.AutoTLS = b(true)
				email(p)
			},
			body: merge(map[string]any{
				"host": "smtp.example.com", "port": float64(587), "username": "user",
				"password": "pass", "encryption": "tls", "autoTLS": true,
			}, emailBody),
		},
		{
			providerType: "resend",
			configure:    func(p *providerResourceModel) { p.APIKey = s("key"); email(p) },
			body:         merge(map[string]any{"apiKey": "key"}, emailBody),
		},
		{
			providerType: "ses",
			configure: func(p *providerResourceModel) {
				p.AccessKey = s("AKIAEXAMPLE")
				p.SecretKey = s("secret")
				p.Region = s("eu-west-1")
				email(p)
			},
			body: merge(map[string]any{"accessKey": "AKIAEXAMPLE", "secretKey": "secret", "region": "eu-west-1"}, emailBody),
		},
		{
			providerType: "twilio",
			configure: func(p *providerResourceModel) {
				p.AccountSid = s("AC123")
				p.AuthToken = s("token")
				p.From = s("+15550100")
			},
			body: map[string]any{"accountSid": "AC123", "authToken": "token", "from": "+15550100"},
		},
		{
			providerType: "vonage",
			configure: func(p *providerResourceModel) {
				p.APIKey = s("key")
				p.APISecret = s("secret")
				p.From = s("+15550100")
			},
			body: map[string]any{"apiKey": "key", "apiSecret": "secret", "from": "+15550100"},
		},
		{
			providerType: "msg91",
			configure: func(p *providerResourceModel) {
				p.SenderID = s("sender")
				p.AuthKey = s("auth")
				p.TemplateID = s("template")
			},
			body: map[string]any{"senderId": "sender", "authKey": "auth", "templateId": "template"},
		},
		{
			providerType: "telesign",
			configure: func(p *providerResourceModel) {
				p.CustomerID = s("customer")
				p.APIKey = s("key")
				p.From = s("+15550100")
			},
			body: map[string]any{"customerId": "customer", "apiKey": "key", "from": "+15550100"},
		},
		{
			providerType: "textmagic",
			configure: func(p *providerResourceModel) {
				p.Username = s("user")
				p.APIKey = s("key")
				p.From = s("+15550100")
			},
			body: map[string]any{"username": "user", "apiKey": "key", "from": "+15550100"},
		},
		{
			providerType: "apns",
			configure: func(p *providerResourceModel) {
				p.AuthKey = s("auth")
				p.AuthKeyID = s("key-id")
				p.TeamID = s("team")
				p.BundleID = s("io.example.app")
				p.Sandbox = b(true)
			},
			body: map[string]any{"authKey": "auth", "authKeyId": "key-id", "teamId": "team", "bundleId": "io.example.app", "sandbox": true},
		},
		{
			providerType: "fcm",
			configure:    func(p *providerResourceModel) { p.ServiceAccount = s(`{"type":"service_account"}`) },
			body:         map[string]any{"serviceAccountJSON": `{"type":"service_account"}`},
		},
		{
			providerType: "appwrite",
			configure:    func(p *providerResourceModel) { p.Qos = i(1); p.Expiry = i(3600) },
			body:         map[string]any{"qos": float64(1), "expiry": float64(3600)},
			// Removing qos from the configuration sends an explicit null,
			// which is what clears it on the server.
			updated:    func(p *providerResourceModel) { p.Qos = types.Int64Null() },
			updateBody: map[string]any{"qos": nil, "expiry": float64(3600)},
		},
	} {
		t.Run(tc.providerType, func(t *testing.T) {
			t.Parallel()
			server, requests := providerServer(t, tc.providerType)
			c := testClient(server.URL)
			messagingClient := appwrite.NewMessaging(c)

			plan := nullModel()
			plan.ID = s("prov-id")
			plan.Name = s("name")
			plan.Type = s(tc.providerType)
			plan.Enabled = b(true)
			tc.configure(&plan)

			if _, err := createProvider(messagingClient, "prov-id", plan); err != nil {
				t.Fatalf("create: %v", err)
			}
			updateBody := tc.body
			if tc.updated != nil {
				tc.updated(&plan)
				updateBody = tc.updateBody
			}
			if _, err := updateProvider(c, messagingClient, plan); err != nil {
				t.Fatalf("update: %v", err)
			}

			if len(*requests) != 2 {
				t.Fatalf("got %d requests, want 2", len(*requests))
			}
			for n, want := range []struct {
				method, path string
				body         map[string]any
			}{
				{http.MethodPost, "/v1/messaging/providers/" + tc.providerType,
					merge(map[string]any{"providerId": "prov-id", "name": "name", "enabled": true}, tc.body)},
				{http.MethodPatch, "/v1/messaging/providers/" + tc.providerType + "/prov-id",
					merge(map[string]any{"name": "name", "enabled": true}, updateBody)},
			} {
				got := (*requests)[n]
				if got.method != want.method || got.path != want.path {
					t.Errorf("request %d: got %s %s, want %s %s", n, got.method, got.path, want.method, want.path)
				}
				if got.project != "project-id" {
					t.Errorf("request %d: X-Appwrite-Project = %q, want project-id", n, got.project)
				}
				if !reflect.DeepEqual(got.body, want.body) {
					t.Errorf("request %d body:\n got %v\nwant %v", n, got.body, want.body)
				}
			}
		})
	}
}

// TestProviderRoutes_omitsUnset checks that an argument left out of the
// configuration is left out of the request too, so the server keeps its own
// default. appwrite is the exception: its update always sends qos and expiry.
func TestProviderRoutes_omitsUnset(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		providerType string
		create       map[string]any
		update       map[string]any
	}{
		{"ses", map[string]any{"providerId": "prov-id", "name": "name"}, map[string]any{"name": "name"}},
		{"appwrite", map[string]any{"providerId": "prov-id", "name": "name"}, map[string]any{"name": "name", "qos": nil, "expiry": nil}},
	} {
		t.Run(tc.providerType, func(t *testing.T) {
			t.Parallel()
			server, requests := providerServer(t, tc.providerType)
			c := testClient(server.URL)
			messagingClient := appwrite.NewMessaging(c)

			plan := nullModel()
			plan.ID = types.StringValue("prov-id")
			plan.Name = types.StringValue("name")
			plan.Type = types.StringValue(tc.providerType)

			if _, err := createProvider(messagingClient, "prov-id", plan); err != nil {
				t.Fatalf("create: %v", err)
			}
			if _, err := updateProvider(c, messagingClient, plan); err != nil {
				t.Fatalf("update: %v", err)
			}
			if got := (*requests)[0].body; !reflect.DeepEqual(got, tc.create) {
				t.Errorf("create body:\n got %v\nwant %v", got, tc.create)
			}
			if got := (*requests)[1].body; !reflect.DeepEqual(got, tc.update) {
				t.Errorf("update body:\n got %v\nwant %v", got, tc.update)
			}
		})
	}
}

func TestProviderRoutes_errors(t *testing.T) {
	t.Parallel()
	server, requests := providerServer(t, "smtp")
	c := testClient(server.URL)
	messagingClient := appwrite.NewMessaging(c)

	plan := nullModel()
	plan.Name = types.StringValue("name")

	plan.Type = types.StringValue("smtp")
	if _, err := createProvider(messagingClient, "prov-id", plan); !errors.Is(err, errSMTPHostRequired) {
		t.Errorf("smtp without host: got %v, want errSMTPHostRequired", err)
	}

	plan.Type = types.StringValue("pigeon")
	if _, err := createProvider(messagingClient, "prov-id", plan); !errors.Is(err, errUnsupportedType) {
		t.Errorf("create: got %v, want errUnsupportedType", err)
	}
	if _, err := updateProvider(c, messagingClient, plan); !errors.Is(err, errUnsupportedType) {
		t.Errorf("update: got %v, want errUnsupportedType", err)
	}

	if len(*requests) != 0 {
		t.Errorf("got %d requests, want none", len(*requests))
	}
}

// TestIsRouteMissing checks the 404 an Appwrite older than 2.3.0 returns for
// the appwrite provider route, an HTML page, is told apart from the JSON 404
// for a provider that does not exist.
func TestIsRouteMissing(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name        string
		contentType string
		body        string
		want        bool
	}{
		{"unknown route", "text/html; charset=UTF-8", "<!DOCTYPE html><html><h1>The page you're looking for doesn't exist.</h1></html>", true},
		{"missing provider", "application/json", `{"message":"Provider with the requested ID could not be found.","code":404,"type":"provider_not_found"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(server.Close)

			plan := nullModel()
			plan.Name = types.StringValue("name")
			plan.Type = types.StringValue("appwrite")
			_, err := createProvider(appwrite.NewMessaging(testClient(server.URL)), "prov-id", plan)
			if err == nil {
				t.Fatal("create succeeded against a 404")
			}
			if got := isRouteMissing(err); got != tc.want {
				t.Errorf("isRouteMissing = %v, want %v (error: %v)", got, tc.want, err)
			}
		})
	}
}

func TestValidateProviderConfig(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		configure func(*providerResourceModel)
		wantError string
	}{
		{"ses arguments on ses", func(p *providerResourceModel) {
			p.Type = types.StringValue("ses")
			p.AccessKey = types.StringValue("a")
			p.SecretKey = types.StringValue("s")
			p.Region = types.StringValue("us-east-1")
		}, ""},
		{"appwrite arguments on appwrite", func(p *providerResourceModel) {
			p.Type = types.StringValue("appwrite")
			p.Qos = types.Int64Value(0)
			p.Expiry = types.Int64Value(60)
		}, ""},
		{"region on sendgrid", func(p *providerResourceModel) {
			p.Type = types.StringValue("sendgrid")
			p.Region = types.StringValue("us-east-1")
		}, "region applies only to ses providers, not sendgrid"},
		{"qos on fcm", func(p *providerResourceModel) {
			p.Type = types.StringValue("fcm")
			p.Qos = types.Int64Value(1)
		}, "qos applies only to appwrite providers, not fcm"},
		{"secret_key on appwrite", func(p *providerResourceModel) {
			p.Type = types.StringValue("appwrite")
			p.SecretKey = types.StringValue("s")
		}, "secret_key applies only to ses providers, not appwrite"},
		{"unknown value is not judged", func(p *providerResourceModel) {
			p.Type = types.StringValue("sendgrid")
			p.Expiry = types.Int64Unknown()
		}, ""},
		// The older arguments are not checked: they have always been
		// accepted on any type.
		{"api_key on ses", func(p *providerResourceModel) {
			p.Type = types.StringValue("ses")
			p.APIKey = types.StringValue("k")
		}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			config := nullModel()
			tc.configure(&config)
			diags := validateProviderConfig(config)
			switch {
			case tc.wantError == "" && diags.HasError():
				t.Errorf("unexpected error: %v", diags)
			case tc.wantError != "" && !diags.HasError():
				t.Errorf("no error, want one containing %q", tc.wantError)
			case tc.wantError != "" && !strings.Contains(diags[0].Detail(), tc.wantError):
				t.Errorf("error %q does not contain %q", diags[0].Detail(), tc.wantError)
			}
		})
	}
}

func TestMapToState_appwriteOptions(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name                string
		provider            string
		options             map[string]any
		wantQos, wantExpiry types.Int64
	}{
		{"set", "appwrite", map[string]any{"qos": float64(1), "expiry": float64(3600)}, types.Int64Value(1), types.Int64Value(3600)},
		{"zero is a value", "appwrite", map[string]any{"qos": float64(0), "expiry": float64(0)}, types.Int64Value(0), types.Int64Value(0)},
		{"null", "appwrite", map[string]any{"qos": nil, "expiry": nil}, types.Int64Null(), types.Int64Null()},
		// Other types never have these options, and state keeps whatever
		// the configuration had, which validation keeps null.
		{"other type", "fcm", map[string]any{}, types.Int64Null(), types.Int64Null()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			model := nullModel()
			(&providerResource{}).mapToState(&models.Provider{Id: "prov-id", Provider: tc.provider, Options: tc.options}, &model)
			if !model.Qos.Equal(tc.wantQos) || !model.Expiry.Equal(tc.wantExpiry) {
				t.Errorf("qos, expiry = %v, %v; want %v, %v", model.Qos, model.Expiry, tc.wantQos, tc.wantExpiry)
			}
		})
	}
}

// nullModel returns a model with every attribute null, as a configuration
// that sets nothing would produce.
func nullModel() providerResourceModel {
	var m providerResourceModel
	v := reflect.ValueOf(&m).Elem()
	for n := 0; n < v.NumField(); n++ {
		switch v.Field(n).Interface().(type) {
		case types.String:
			v.Field(n).Set(reflect.ValueOf(types.StringNull()))
		case types.Bool:
			v.Field(n).Set(reflect.ValueOf(types.BoolNull()))
		case types.Int64:
			v.Field(n).Set(reflect.ValueOf(types.Int64Null()))
		}
	}
	return m
}

func merge(maps ...map[string]any) map[string]any {
	out := map[string]any{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}
