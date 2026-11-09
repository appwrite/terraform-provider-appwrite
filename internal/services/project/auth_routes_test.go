package project

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The tests here drive each resource's CRUD methods against a fake of the
// project API and check the route and body of every write. Unit tests cannot
// see the schema defaults Terraform applies at plan time, so they pass fully
// populated plans; the acceptance tests cover planning.

func checkWrites(t *testing.T, got []authRequest, want []authRequest) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d writes, want %d:\n got  %v\n want %v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i].method != want[i].method || got[i].path != want[i].path {
			t.Errorf("write %d: got %s %s, want %s %s", i, got[i].method, got[i].path, want[i].method, want[i].path)
		}
		if !reflect.DeepEqual(got[i].body, want[i].body) {
			t.Errorf("write %d to %s: body %v, want %v", i, want[i].path, got[i].body, want[i].body)
		}
	}
}

func sameState(t *testing.T, label string, got, want tfsdk.State) {
	t.Helper()
	if !got.Raw.Equal(want.Raw) {
		t.Errorf("%s: state differs\n got  %v\n want %v", label, got.Raw, want.Raw)
	}
}

func authMethodsPlan(enabled map[string]bool) authMethodsResourceModel {
	m := authMethodsResourceModel{ID: types.StringUnknown(), ProjectID: types.StringUnknown()}
	for methodID, field := range m.fields() {
		*field = types.BoolValue(enabled[methodID])
	}
	return m
}

func TestAuthMethodRoutes(t *testing.T) {
	t.Parallel()
	fake, server := newFakeAuthServer(t)
	c := newCRUD(t, NewAuthMethodsResource(), testClients(server.URL))

	// Every method off: each has its own route.
	state, errText := c.create(authMethodsPlan(map[string]bool{}))
	if errText != "" {
		t.Fatalf("create: %s", errText)
	}
	want := make([]authRequest, 0, len(authMethods))
	for _, method := range authMethods {
		want = append(want, authRequest{http.MethodPatch, "/v1/project/auth-methods/" + method.id, map[string]any{"enabled": false}})
	}
	checkWrites(t, fake.writes(), want)

	var created authMethodsResourceModel
	state.Get(context.Background(), &created)
	if created.ID.ValueString() != fakeProjectID || created.ProjectID.ValueString() != fakeProjectID {
		t.Errorf("id %s, project_id %s, want both %s", created.ID, created.ProjectID, fakeProjectID)
	}

	// Changing one method writes only that one.
	fake.resetRequests()
	state, errText = c.update(created, authMethodsPlan(map[string]bool{"jwt": true}))
	if errText != "" {
		t.Fatalf("update: %s", errText)
	}
	checkWrites(t, fake.writes(), []authRequest{
		{http.MethodPatch, "/v1/project/auth-methods/jwt", map[string]any{"enabled": true}},
	})

	sameState(t, "import", c.imported(fakeProjectID), state)

	// Destroy enables the six still off.
	fake.resetRequests()
	c.delete(state)
	want = nil
	for _, method := range authMethods {
		if method.id != "jwt" {
			want = append(want, authRequest{http.MethodPatch, "/v1/project/auth-methods/" + method.id, map[string]any{"enabled": true}})
		}
	}
	checkWrites(t, fake.writes(), want)
	for id, enabled := range fake.current().methods {
		if !enabled {
			t.Errorf("%s still disabled after destroy", id)
		}
	}
}

func authPoliciesPlan(t *testing.T, p authPolicies) authPoliciesResourceModel {
	t.Helper()
	m := authPoliciesResourceModel{ID: types.StringUnknown(), ProjectID: types.StringUnknown()}
	if diags := policiesToModel(context.Background(), p, &m); diags.HasError() {
		t.Fatalf("model: %v", diags)
	}
	return m
}

// TestAuthPolicyRoutes changes one policy at a time from the defaults and
// checks that exactly that policy's route is written, with every field it
// takes, and that destroy writes the defaults back to the same route.
func TestAuthPolicyRoutes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		id     string
		change func(*authPolicies)
		body   map[string]any
		reset  map[string]any
	}{
		{"password-dictionary", func(p *authPolicies) { p.PasswordDictionary = true },
			map[string]any{"enabled": true}, map[string]any{"enabled": false}},
		// 0 is not a length the route accepts; null is what disables it.
		{"password-history", func(p *authPolicies) { p.PasswordHistory = 5 },
			map[string]any{"total": 5.0}, map[string]any{"total": nil}},
		{"password-strength", func(p *authPolicies) {
			p.PasswordStrength = passwordStrengthPolicy{Min: 12, Uppercase: true, Number: true}
		},
			map[string]any{"min": 12.0, "uppercase": true, "lowercase": false, "number": true, "symbols": false},
			map[string]any{"min": 8.0, "uppercase": false, "lowercase": false, "number": false, "symbols": false}},
		{"password-personal-data", func(p *authPolicies) { p.PasswordPersonalData = true },
			map[string]any{"enabled": true}, map[string]any{"enabled": false}},
		{"password-pwned", func(p *authPolicies) { p.PasswordPwned = passwordPwnedPolicy{Enabled: true, Users: true} },
			map[string]any{"enabled": true, "sessions": false, "users": true},
			map[string]any{"enabled": true, "sessions": false, "users": false}},
		{"session-alert", func(p *authPolicies) { p.SessionAlert = true },
			map[string]any{"enabled": true}, map[string]any{"enabled": false}},
		{"session-duration", func(p *authPolicies) { p.SessionDuration = 3600 },
			map[string]any{"duration": 3600.0}, map[string]any{"duration": 31536000.0}},
		{"session-invalidation", func(p *authPolicies) { p.SessionInvalidation = false },
			map[string]any{"enabled": false}, map[string]any{"enabled": true}},
		{"session-limit", func(p *authPolicies) { p.SessionLimit = 50 },
			map[string]any{"total": 50.0}, map[string]any{"total": 10.0}},
		{"user-limit", func(p *authPolicies) { p.UserLimit = 100 },
			map[string]any{"total": 100.0}, map[string]any{"total": 0.0}},
		{"membership-privacy", func(p *authPolicies) {
			p.MembershipPrivacy = membershipPrivacyPolicy{UserID: true, UserName: true, UserAccessedAt: true}
		},
			map[string]any{"userId": true, "userEmail": false, "userPhone": false, "userName": true, "userMFA": false, "userAccessedAt": true},
			map[string]any{"userId": false, "userEmail": false, "userPhone": false, "userName": false, "userMFA": false, "userAccessedAt": false}},
		{"mfa-factors", func(p *authPolicies) { p.MFAFactors = mfaFactorsPolicy{TOTP: true, Custom: true} },
			map[string]any{"totp": true, "email": false, "phone": false, "custom": true},
			map[string]any{"totp": true, "email": true, "phone": true, "custom": false}},
		{"deny-aliased-email", func(p *authPolicies) { p.DenyAliasedEmail = true },
			map[string]any{"enabled": true}, map[string]any{"enabled": false}},
		{"deny-disposable-email", func(p *authPolicies) { p.DenyDisposableEmail = true },
			map[string]any{"enabled": true}, map[string]any{"enabled": false}},
		{"deny-free-email", func(p *authPolicies) { p.DenyFreeEmail = true },
			map[string]any{"enabled": true}, map[string]any{"enabled": false}},
		{"deny-corporate-email", func(p *authPolicies) { p.DenyCorporateEmail = true },
			map[string]any{"enabled": true}, map[string]any{"enabled": false}},
	}

	// A policy added to the resource without a case here fails, rather than
	// going untested.
	if len(cases) != len(policies) {
		t.Fatalf("%d cases for %d policies", len(cases), len(policies))
	}
	for i, p := range policies {
		if cases[i].id != p.id {
			t.Fatalf("case %d is %s, policy %d is %s", i, cases[i].id, i, p.id)
		}
	}

	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			t.Parallel()
			fake, server := newFakeAuthServer(t)
			fake.supportPasswordPwned()
			c := newCRUD(t, NewAuthPoliciesResource(), testClients(server.URL))

			want := defaultAuthPolicies
			tc.change(&want)
			state, errText := c.create(authPoliciesPlan(t, want))
			if errText != "" {
				t.Fatalf("create: %s", errText)
			}
			route := "/v1/project/policies/" + tc.id
			checkWrites(t, fake.writes(), []authRequest{{http.MethodPatch, route, tc.body}})

			// What Create stored is what a refresh reads back.
			refreshed, _ := c.read(state)
			sameState(t, "refresh", refreshed, state)

			fake.resetRequests()
			c.delete(state)
			checkWrites(t, fake.writes(), []authRequest{{http.MethodPatch, route, tc.reset}})
		})
	}
}

func TestAuthPoliciesUnchangedSendsNothing(t *testing.T) {
	t.Parallel()
	fake, server := newFakeAuthServer(t)
	fake.supportPasswordPwned()
	c := newCRUD(t, NewAuthPoliciesResource(), testClients(server.URL))

	state, errText := c.create(authPoliciesPlan(t, defaultAuthPolicies))
	if errText != "" {
		t.Fatalf("create: %s", errText)
	}
	c.delete(state)
	checkWrites(t, fake.writes(), nil)
}

// Every policy changed at once, then imported: the imported state must match
// the created one, or the first plan after an import would show a diff.
func TestAuthPoliciesImport(t *testing.T) {
	t.Parallel()
	fake, server := newFakeAuthServer(t)
	fake.supportPasswordPwned()
	c := newCRUD(t, NewAuthPoliciesResource(), testClients(server.URL))

	want := authPolicies{
		PasswordDictionary:   true,
		PasswordHistory:      20,
		PasswordPersonalData: true,
		PasswordStrength:     passwordStrengthPolicy{Min: 256, Uppercase: true, Lowercase: true, Number: true, Symbols: true},
		PasswordPwned:        passwordPwnedPolicy{Sessions: true, Users: true},
		SessionAlert:         true,
		SessionDuration:      60,
		SessionLimit:         100,
		UserLimit:            10000,
		MembershipPrivacy:    membershipPrivacyPolicy{true, true, true, true, true, true},
		MFAFactors:           mfaFactorsPolicy{Custom: true},
		DenyAliasedEmail:     true,
		DenyDisposableEmail:  true,
		DenyFreeEmail:        true,
		DenyCorporateEmail:   true,
	}
	state, errText := c.create(authPoliciesPlan(t, want))
	if errText != "" {
		t.Fatalf("create: %s", errText)
	}
	if got := len(fake.writes()); got != len(policies) {
		t.Errorf("got %d writes, want one per policy (%d)", got, len(policies))
	}
	sameState(t, "import", c.imported(fakeProjectID), state)
}

// Appwrite 2.2.0 has no password-pwned policy. Leaving it at its defaults has
// to work there; asking for anything else has to fail before any other
// policy is written.
func TestAuthPoliciesOnServerWithoutPasswordPwned(t *testing.T) {
	t.Parallel()
	fake, server := newFakeAuthServer(t)
	c := newCRUD(t, NewAuthPoliciesResource(), testClients(server.URL))

	want := defaultAuthPolicies
	want.SessionLimit = 50
	state, errText := c.create(authPoliciesPlan(t, want))
	if errText != "" {
		t.Fatalf("create: %s", errText)
	}
	checkWrites(t, fake.writes(), []authRequest{
		{http.MethodPatch, "/v1/project/policies/session-limit", map[string]any{"total": 50.0}},
	})
	sameState(t, "import", c.imported(fakeProjectID), state)

	var prior authPoliciesResourceModel
	state.Get(context.Background(), &prior)
	fake.resetRequests()
	want.SessionAlert = true
	want.PasswordPwned.Users = true
	_, errText = c.update(prior, authPoliciesPlan(t, want))
	if !strings.Contains(errText, "password-pwned") || !strings.Contains(errText, "not supported") {
		t.Errorf("update error %q, want password-pwned reported as not supported", errText)
	}
	checkWrites(t, fake.writes(), nil)

	c.delete(state)
	checkWrites(t, fake.writes(), []authRequest{
		{http.MethodPatch, "/v1/project/policies/session-limit", map[string]any{"total": 10.0}},
	})
}

func mockPhone(number, otp string) mockPhoneResourceModel {
	return mockPhoneResourceModel{
		ID:        types.StringUnknown(),
		ProjectID: types.StringUnknown(),
		Number:    types.StringValue(number),
		OTP:       types.StringValue(otp),
		CreatedAt: types.StringUnknown(),
		UpdatedAt: types.StringUnknown(),
	}
}

func TestMockPhoneRoutes(t *testing.T) {
	t.Parallel()
	fake, server := newFakeAuthServer(t)
	c := newCRUD(t, NewMockPhoneResource(), testClients(server.URL))

	state, errText := c.create(mockPhone("+15555550100", "123456"))
	if errText != "" {
		t.Fatalf("create: %s", errText)
	}
	var created mockPhoneResourceModel
	state.Get(context.Background(), &created)
	if created.ID.ValueString() != "+15555550100" {
		t.Errorf("id %s, want the number", created.ID)
	}

	plan := created
	plan.OTP = types.StringValue("654321")
	plan.UpdatedAt = types.StringUnknown()
	state, errText = c.update(created, plan)
	if errText != "" {
		t.Fatalf("update: %s", errText)
	}

	sameState(t, "import by number", c.imported("+15555550100"), state)
	sameState(t, "import by project and number", c.imported(fakeProjectID+"/+15555550100"), state)

	c.delete(state)
	checkWrites(t, fake.writes(), []authRequest{
		{http.MethodPost, "/v1/project/mock-phones", map[string]any{"number": "+15555550100", "otp": "123456"}},
		// The + travels percent-encoded in the path.
		{http.MethodPut, "/v1/project/mock-phones/%2B15555550100", map[string]any{"otp": "654321"}},
		{http.MethodDelete, "/v1/project/mock-phones/%2B15555550100", map[string]any{}},
	})

	if _, exists := c.read(state); exists {
		t.Error("read after delete kept the resource")
	}
}

func TestMockPhoneAlreadyExists(t *testing.T) {
	t.Parallel()
	_, server := newFakeAuthServer(t)
	c := newCRUD(t, NewMockPhoneResource(), testClients(server.URL))

	if _, errText := c.create(mockPhone("+15555550100", "123456")); errText != "" {
		t.Fatalf("create: %s", errText)
	}
	_, errText := c.create(mockPhone("+15555550100", "123456"))
	if !strings.Contains(errText, "already a mock number") || !strings.Contains(errText, "terraform import") {
		t.Errorf("duplicate create: %q, want guidance to import", errText)
	}
}

func TestImportIDs(t *testing.T) {
	t.Parallel()
	_, server := newFakeAuthServer(t)
	clients := testClients(server.URL)

	for _, tc := range []struct {
		name string
		crud *crud
		id   string
	}{
		{"methods with slash", newCRUD(t, NewAuthMethodsResource(), clients), "a/b"},
		{"policies empty", newCRUD(t, NewAuthPoliciesResource(), clients), ""},
		{"mock phone three parts", newCRUD(t, NewMockPhoneResource(), clients), "a/b/c"},
		{"mock phone empty number", newCRUD(t, NewMockPhoneResource(), clients), "a/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !tc.crud.importRefused(tc.id) {
				t.Errorf("import of %q was accepted", tc.id)
			}
		})
	}
}
