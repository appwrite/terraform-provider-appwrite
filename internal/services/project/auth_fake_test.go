package project

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

const fakeProjectID = "fake-project"

type authRequest struct {
	method string
	// path is the escaped path, so a test can see how a phone number's + was
	// sent.
	path string
	body map[string]any
}

// fakeAuthServer stands in for the parts of the Appwrite 2.2.0 project API the
// auth resources use, with the behavior observed against a real server:
//
//   - every write replaces the whole policy, so an update body carries every
//     field the policy has;
//   - password-history rejects 0 and disables the history on null;
//   - mock phone paths accept the number with its + percent-encoded;
//   - a write reads the project's auth settings, changes them and writes all
//     of them back, so a write landing in between is lost.
//
// It records every request. writeDelay widens the window between a write's
// read and its write-back, which is what lets a test show lost updates.
type fakeAuthServer struct {
	t *testing.T

	mu       sync.Mutex
	state    *authState
	requests []authRequest

	writeDelay time.Duration
	inFlight   atomic.Int32
	overlapped atomic.Bool
}

// authState is the project's auth settings: the one attribute of the project
// document that every route here reads and rewrites.
type authState struct {
	methods  map[string]bool
	policies map[string]map[string]any
	order    []string
	mocks    map[string]map[string]any
}

func (s *authState) clone() *authState {
	c := &authState{
		methods:  make(map[string]bool, len(s.methods)),
		policies: make(map[string]map[string]any, len(s.policies)),
		order:    append([]string(nil), s.order...),
		mocks:    make(map[string]map[string]any, len(s.mocks)),
	}
	for k, v := range s.methods {
		c.methods[k] = v
	}
	for k, v := range s.policies {
		c.policies[k] = copyFields(v)
	}
	for k, v := range s.mocks {
		c.mocks[k] = copyFields(v)
	}
	return c
}

func copyFields(m map[string]any) map[string]any {
	c := make(map[string]any, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

func newFakeAuthServer(t *testing.T) (*fakeAuthServer, *httptest.Server) {
	t.Helper()
	f := &fakeAuthServer{
		t: t,
		state: &authState{
			methods:  map[string]bool{},
			policies: map[string]map[string]any{},
			mocks:    map[string]map[string]any{},
		},
	}
	for _, method := range authMethods {
		f.state.methods[method.id] = true
	}
	// The policies a new 2.2.0 project reports, in its order. password-pwned
	// is added by tests that need a 2.3.0 server.
	for _, p := range []struct {
		id     string
		fields map[string]any
	}{
		{"password-dictionary", map[string]any{"enabled": false}},
		{"password-history", map[string]any{"total": 0.0}},
		{"password-strength", map[string]any{"min": 8.0, "uppercase": false, "lowercase": false, "number": false, "symbols": false}},
		{"password-personal-data", map[string]any{"enabled": false}},
		{"session-alert", map[string]any{"enabled": false}},
		{"session-duration", map[string]any{"duration": 31536000.0}},
		{"session-invalidation", map[string]any{"enabled": true}},
		{"session-limit", map[string]any{"total": 10.0}},
		{"user-limit", map[string]any{"total": 0.0}},
		{"membership-privacy", map[string]any{"userId": false, "userEmail": false, "userPhone": false, "userName": false, "userMFA": false, "userAccessedAt": false}},
		{"mfa-factors", map[string]any{"totp": true, "email": true, "phone": true, "custom": false}},
		{"deny-aliased-email", map[string]any{"enabled": false}},
		{"deny-disposable-email", map[string]any{"enabled": false}},
		{"deny-free-email", map[string]any{"enabled": false}},
		{"deny-corporate-email", map[string]any{"enabled": false}},
	} {
		f.addPolicy(p.id, p.fields)
	}

	server := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(server.Close)
	return f, server
}

func (f *fakeAuthServer) addPolicy(id string, fields map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state.policies[id] = fields
	f.state.order = append(f.state.order, id)
}

// supportPasswordPwned makes the fake a 2.3.0 server.
func (f *fakeAuthServer) supportPasswordPwned() {
	f.addPolicy(passwordPwnedPolicyID, map[string]any{"enabled": true, "sessions": false, "users": false})
}

// current returns a copy of the stored settings.
func (f *fakeAuthServer) current() *authState {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state.clone()
}

// writes returns the requests that changed something, in order.
func (f *fakeAuthServer) writes() []authRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	var writes []authRequest
	for _, r := range f.requests {
		if r.method != http.MethodGet {
			writes = append(writes, r)
		}
	}
	return writes
}

func (f *fakeAuthServer) resetRequests() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = nil
}

func (f *fakeAuthServer) serve(w http.ResponseWriter, r *http.Request) {
	body := map[string]any{}
	_ = json.NewDecoder(r.Body).Decode(&body)

	f.mu.Lock()
	f.requests = append(f.requests, authRequest{method: r.Method, path: r.URL.EscapedPath(), body: body})
	if r.Method == http.MethodGet {
		defer f.mu.Unlock()
		f.handle(f.state, w, r, body)
		return
	}

	// Read the settings, change them, write all of them back, with nothing to
	// stop another write from landing in between.
	working := f.state.clone()
	f.mu.Unlock()

	if f.inFlight.Add(1) > 1 {
		f.overlapped.Store(true)
	}
	time.Sleep(f.writeDelay)
	recorder := httptest.NewRecorder()
	f.handle(working, recorder, r, body)
	f.inFlight.Add(-1)

	if recorder.Code < http.StatusMultipleChoices {
		f.mu.Lock()
		f.state = working
		f.mu.Unlock()
	}
	for key, values := range recorder.Header() {
		w.Header()[key] = values
	}
	w.WriteHeader(recorder.Code)
	_, _ = w.Write(recorder.Body.Bytes())
}

func (f *fakeAuthServer) handle(s *authState, w http.ResponseWriter, r *http.Request, body map[string]any) {
	path := strings.TrimPrefix(r.URL.Path, "/v1")
	switch {
	case r.Method == http.MethodGet && path == "/project":
		f.respond(w, http.StatusOK, s.project())

	case r.Method == http.MethodPatch && strings.HasPrefix(path, "/project/auth-methods/"):
		id := strings.TrimPrefix(path, "/project/auth-methods/")
		if _, ok := s.methods[id]; !ok {
			f.fail(w, http.StatusBadRequest, "general_argument_invalid")
			return
		}
		enabled, ok := body["enabled"].(bool)
		if !ok {
			f.fail(w, http.StatusBadRequest, "general_argument_invalid")
			return
		}
		s.methods[id] = enabled
		f.respond(w, http.StatusOK, s.project())

	case r.Method == http.MethodGet && path == "/project/policies":
		list := make([]map[string]any, 0, len(s.order))
		for _, id := range s.order {
			list = append(list, s.policy(id))
		}
		f.respond(w, http.StatusOK, map[string]any{"total": len(list), "policies": list})

	case r.Method == http.MethodPatch && strings.HasPrefix(path, "/project/policies/"):
		id := strings.TrimPrefix(path, "/project/policies/")
		policy, ok := s.policies[id]
		if !ok {
			// An older server has no route for a newer policy.
			f.fail(w, http.StatusNotFound, "general_route_not_found")
			return
		}
		update := copyFields(body)
		if id == "password-history" {
			total, isNumber := body["total"].(float64)
			switch {
			case body["total"] == nil:
				update["total"] = 0.0
			case !isNumber || total < 1:
				f.fail(w, http.StatusBadRequest, "general_argument_invalid")
				return
			}
		}
		for key, value := range update {
			if _, known := policy[key]; !known {
				f.t.Errorf("PATCH %s: unexpected field %q", path, key)
			}
			policy[key] = value
		}
		if id == "password-strength" {
			f.respond(w, http.StatusOK, s.policy(id))
			return
		}
		f.respond(w, http.StatusOK, s.project())

	case r.Method == http.MethodPost && path == "/project/mock-phones":
		number, _ := body["number"].(string)
		if _, exists := s.mocks[number]; exists {
			f.fail(w, http.StatusConflict, "mock_number_already_exists")
			return
		}
		s.mocks[number] = map[string]any{"number": number, "otp": body["otp"], "$createdAt": "2026-01-01T00:00:00.000+00:00", "$updatedAt": "2026-01-01T00:00:00.000+00:00"}
		f.respond(w, http.StatusCreated, s.mocks[number])

	case strings.HasPrefix(path, "/project/mock-phones/"):
		number := strings.TrimPrefix(path, "/project/mock-phones/")
		mock, exists := s.mocks[number]
		if !exists {
			f.fail(w, http.StatusNotFound, "mock_number_not_found")
			return
		}
		switch r.Method {
		case http.MethodGet:
			f.respond(w, http.StatusOK, mock)
		case http.MethodPut:
			mock["otp"] = body["otp"]
			mock["$updatedAt"] = "2026-01-02T00:00:00.000+00:00"
			f.respond(w, http.StatusOK, mock)
		case http.MethodDelete:
			delete(s.mocks, number)
			w.WriteHeader(http.StatusNoContent)
		}

	default:
		f.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		f.fail(w, http.StatusNotFound, "general_route_not_found")
	}
}

func (s *authState) project() map[string]any {
	methods := make([]map[string]any, 0, len(authMethods))
	for _, method := range authMethods {
		methods = append(methods, map[string]any{"$id": method.id, "enabled": s.methods[method.id]})
	}
	return map[string]any{"$id": fakeProjectID, "name": "Fake", "authMethods": methods}
}

func (s *authState) policy(id string) map[string]any {
	out := copyFields(s.policies[id])
	out["$id"] = id
	return out
}

func (f *fakeAuthServer) respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (f *fakeAuthServer) fail(w http.ResponseWriter, status int, errorType string) {
	f.respond(w, status, map[string]any{"message": errorType, "code": status, "type": errorType})
}

// testClients configures the resources to talk to the fake with a standard
// key.
func testClients(url string) *common.AppwriteClients {
	return &common.AppwriteClients{
		BaseOptions: []client.ClientOption{
			appwrite.WithEndpoint(url + "/v1"),
			appwrite.WithKey("standard_test"),
		},
		ProjectCredentialType: common.CredentialTypeStandard,
		ProjectID:             fakeProjectID,
	}
}

// crud drives a resource's CRUD methods directly, the way Terraform would
// after planning, so a test can check what each one sends.
type crud struct {
	t        *testing.T
	resource resource.Resource
	schema   resource.SchemaResponse
}

func newCRUD(t *testing.T, r resource.Resource, clients *common.AppwriteClients) *crud {
	t.Helper()
	configurable, ok := r.(resource.ResourceWithConfigure)
	if !ok {
		t.Fatalf("%T cannot be configured", r)
	}
	configurable.Configure(context.Background(), resource.ConfigureRequest{ProviderData: clients}, &resource.ConfigureResponse{})
	c := &crud{t: t, resource: r}
	r.Schema(context.Background(), resource.SchemaRequest{}, &c.schema)
	if c.schema.Diagnostics.HasError() {
		t.Fatalf("schema: %v", c.schema.Diagnostics)
	}
	return c
}

func (c *crud) plan(model any) tfsdk.Plan {
	c.t.Helper()
	plan := tfsdk.Plan{Schema: c.schema.Schema}
	if diags := plan.Set(context.Background(), model); diags.HasError() {
		c.t.Fatalf("plan: %v", diags)
	}
	return plan
}

func (c *crud) state(model any) tfsdk.State {
	c.t.Helper()
	state := tfsdk.State{Schema: c.schema.Schema}
	if diags := state.Set(context.Background(), model); diags.HasError() {
		c.t.Fatalf("state: %v", diags)
	}
	return state
}

// create returns the state Create wrote, or the diagnostics' error text.
func (c *crud) create(model any) (tfsdk.State, string) {
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: c.schema.Schema}}
	c.resource.Create(context.Background(), resource.CreateRequest{Plan: c.plan(model)}, resp)
	return resp.State, diagnosticsText(resp.Diagnostics.Errors())
}

func (c *crud) update(prior, model any) (tfsdk.State, string) {
	resp := &resource.UpdateResponse{State: c.state(prior)}
	c.resource.Update(context.Background(), resource.UpdateRequest{Plan: c.plan(model), State: c.state(prior)}, resp)
	return resp.State, diagnosticsText(resp.Diagnostics.Errors())
}

// read returns the refreshed state and whether the resource is still there.
func (c *crud) read(state tfsdk.State) (tfsdk.State, bool) {
	c.t.Helper()
	resp := &resource.ReadResponse{State: state}
	c.resource.Read(context.Background(), resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		c.t.Fatalf("read: %v", resp.Diagnostics)
	}
	return resp.State, !resp.State.Raw.IsNull()
}

func (c *crud) delete(state tfsdk.State) {
	c.t.Helper()
	resp := &resource.DeleteResponse{}
	c.resource.Delete(context.Background(), resource.DeleteRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		c.t.Fatalf("delete: %v", resp.Diagnostics)
	}
}

// imported returns the state an import of id followed by a refresh produces.
func (c *crud) imported(id string) tfsdk.State {
	c.t.Helper()
	ctx := context.Background()
	resp := &resource.ImportStateResponse{State: tfsdk.State{
		Schema: c.schema.Schema,
		Raw:    tftypes.NewValue(c.schema.Schema.Type().TerraformType(ctx), nil),
	}}
	c.importer().ImportState(ctx, resource.ImportStateRequest{ID: id}, resp)
	if resp.Diagnostics.HasError() {
		c.t.Fatalf("import: %v", resp.Diagnostics)
	}
	state, ok := c.read(resp.State)
	if !ok {
		c.t.Fatalf("import of %s read nothing", id)
	}
	return state
}

// importRefused reports whether importing id was refused.
func (c *crud) importRefused(id string) bool {
	ctx := context.Background()
	resp := &resource.ImportStateResponse{State: tfsdk.State{
		Schema: c.schema.Schema,
		Raw:    tftypes.NewValue(c.schema.Schema.Type().TerraformType(ctx), nil),
	}}
	c.importer().ImportState(ctx, resource.ImportStateRequest{ID: id}, resp)
	return resp.Diagnostics.HasError()
}

func (c *crud) importer() resource.ResourceWithImportState {
	c.t.Helper()
	importer, ok := c.resource.(resource.ResourceWithImportState)
	if !ok {
		c.t.Fatalf("%T cannot be imported", c.resource)
	}
	return importer
}

func diagnosticsText(diags diag.Diagnostics) string {
	parts := make([]string, 0, len(diags))
	for _, d := range diags {
		parts = append(parts, d.Summary()+": "+d.Detail())
	}
	return strings.Join(parts, "; ")
}
