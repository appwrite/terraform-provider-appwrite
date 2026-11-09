package project

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func decodeBody(t *testing.T, raw string) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	body := map[string]any{}
	if err := decoder.Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body
}

// fullModel sets every argument the provider takes.
func fullModel(p oauthProvider) oauthProviderModel {
	m := emptyModel(p)
	m.Enabled = types.BoolValue(true)
	m.ClientID = types.StringValue("client-" + p.id)
	for _, name := range p.extras {
		switch v := m.extra(name).(type) {
		case *types.String:
			*v = types.StringValue(name + "-value")
		case *types.Bool:
			*v = types.BoolValue(true)
		case *types.Int64:
			*v = types.Int64Value(3600)
		case *types.Set:
			*v = stringSet(name+"-b", name+"-a")
		}
	}
	return m
}

// emptyModel is a configuration naming only the provider.
func emptyModel(p oauthProvider) oauthProviderModel {
	m := oauthProviderModel{
		ProviderID:            types.StringValue(p.id),
		Enabled:               types.BoolValue(true),
		ClientSecret:          types.StringNull(),
		ClientSecretWO:        types.StringNull(),
		ClientSecretWOVersion: types.Int64Null(),
		ClientID:              types.StringNull(),
		ProjectID:             types.StringNull(),
	}
	for name, extra := range oauthExtras {
		clearField(m.extra(name))
		if !extra.serverDefault {
			continue
		}
		// Computed arguments the configuration leaves unset are unknown at
		// apply.
		switch v := m.extra(name).(type) {
		case *types.Int64:
			*v = types.Int64Unknown()
		case *types.Set:
			*v = types.SetUnknown(types.StringType)
		}
	}
	return m
}

func stringSet(values ...string) types.Set {
	elements := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elements = append(elements, types.StringValue(v))
	}
	return types.SetValueMust(types.StringType, elements)
}

// The responses below are what Appwrite 2.2.0 returns for these providers.
func TestApplyOAuthResponse(t *testing.T) {
	t.Parallel()

	t.Run("the secret is kept, an unset argument stays null", func(t *testing.T) {
		t.Parallel()
		p, _ := lookupOAuthProvider("auth0")
		m := emptyModel(p)
		m.ClientSecret = types.StringValue("configured")
		applyOAuthResponse(p, decodeBody(t, `{"$id":"auth0","enabled":true,"clientId":"abc","clientSecret":"","endpoint":""}`), &m)

		if m.ClientSecret.ValueString() != "configured" {
			t.Errorf("client_secret = %s, want the configured value kept", m.ClientSecret)
		}
		if m.ClientID.ValueString() != "abc" || !m.Enabled.ValueBool() || m.ID.ValueString() != "auth0" {
			t.Errorf("got client_id %s, enabled %s, id %s", m.ClientID, m.Enabled, m.ID)
		}
		if !m.Endpoint.IsNull() {
			t.Errorf("endpoint = %s, want null for an argument the configuration leaves unset", m.Endpoint)
		}
		// Appwrite 2.2.0 does not return prompt for auth0, which newer
		// versions accept. Unknown at apply, it resolves to null.
		if !m.Prompt.IsNull() {
			t.Errorf("prompt = %s, want null", m.Prompt)
		}
	})

	t.Run("a value cleared outside Terraform shows as drift", func(t *testing.T) {
		t.Parallel()
		p, _ := lookupOAuthProvider("keycloak")
		m := emptyModel(p)
		m.ClientID = types.StringValue("abc")
		m.Endpoint = types.StringValue("keycloak.example.com")
		m.RealmName = types.StringValue("main")
		applyOAuthResponse(p, decodeBody(t, `{"$id":"keycloak","enabled":false,"clientId":"","clientSecret":"","endpoint":"keycloak.example.com","realmName":""}`), &m)

		if m.ClientID.IsNull() || m.ClientID.ValueString() != "" {
			t.Errorf("client_id = %s, want the empty string", m.ClientID)
		}
		if m.RealmName.IsNull() || m.RealmName.ValueString() != "" {
			t.Errorf("realm_name = %s, want the empty string", m.RealmName)
		}
		if m.Enabled.ValueBool() {
			t.Error("enabled = true, want false")
		}
	})

	t.Run("a setting the server does not return keeps the configured value", func(t *testing.T) {
		t.Parallel()
		p, _ := lookupOAuthProvider("github")
		m := emptyModel(p)
		m.Prompt = stringSet("consent")
		applyOAuthResponse(p, decodeBody(t, `{"$id":"github","enabled":true,"clientId":"abc","clientSecret":""}`), &m)

		if !m.Prompt.Equal(stringSet("consent")) {
			t.Errorf("prompt = %s, want the configured value kept", m.Prompt)
		}
	})

	t.Run("Apple key and team IDs survive a server that blanks them", func(t *testing.T) {
		t.Parallel()
		p, _ := lookupOAuthProvider("apple")
		m := emptyModel(p)
		m.KeyID = types.StringValue("P4000000N8")
		m.TeamID = types.StringValue("D4000000R6")
		m.NativeClientIDs = stringSet("com.example.app")
		applyOAuthResponse(p, decodeBody(t, `{"$id":"apple","enabled":false,"serviceId":"com.example.web","keyId":"","teamId":"","p8File":"","nativeEnabled":false,"nativeClientIds":["com.example.app"]}`), &m)

		if m.KeyID.ValueString() != "P4000000N8" || m.TeamID.ValueString() != "D4000000R6" {
			t.Errorf("key_id %s, team_id %s, want the configured values kept", m.KeyID, m.TeamID)
		}
		if m.ClientID.ValueString() != "com.example.web" {
			t.Errorf("client_id = %s, want the Services ID", m.ClientID)
		}
		if !m.NativeEnabled.IsNull() {
			t.Errorf("native_enabled = %s, want null for false when unset", m.NativeEnabled)
		}
		if !m.NativeClientIDs.Equal(stringSet("com.example.app")) {
			t.Errorf("native_client_ids = %s", m.NativeClientIDs)
		}
	})

	t.Run("Apple key and team IDs are read when returned", func(t *testing.T) {
		t.Parallel()
		p, _ := lookupOAuthProvider("apple")
		m := emptyModel(p)
		applyOAuthResponse(p, decodeBody(t, `{"$id":"apple","enabled":false,"serviceId":"","keyId":"P4000000N8","teamId":"D4000000R6","p8File":"","nativeEnabled":false,"nativeClientIds":[]}`), &m)

		if m.KeyID.ValueString() != "P4000000N8" || m.TeamID.ValueString() != "D4000000R6" {
			t.Errorf("key_id %s, team_id %s, want the returned values, as after an import", m.KeyID, m.TeamID)
		}
		if !m.NativeClientIDs.IsNull() {
			t.Errorf("native_client_ids = %s, want null for an empty list when unset", m.NativeClientIDs)
		}
	})

	t.Run("server defaults fill unset computed arguments", func(t *testing.T) {
		t.Parallel()
		p, _ := lookupOAuthProvider("google")
		m := emptyModel(p)
		applyOAuthResponse(p, decodeBody(t, `{"$id":"google","enabled":false,"clientId":"","clientSecret":"","prompt":["consent"],"nativeEnabled":false,"nativeClientIds":[]}`), &m)

		if !m.Prompt.Equal(stringSet("consent")) {
			t.Errorf("prompt = %s, want Google's default", m.Prompt)
		}
	})

	t.Run("OIDC max_age and an empty prompt", func(t *testing.T) {
		t.Parallel()
		p, _ := lookupOAuthProvider("oidc")
		m := emptyModel(p)
		m.WellKnownURL = types.StringValue("https://idp.example.com/.well-known/openid-configuration")
		applyOAuthResponse(p, decodeBody(t, `{"$id":"oidc","enabled":true,"clientId":"abc","clientSecret":"","wellKnownURL":"https://idp.example.com/.well-known/openid-configuration","authorizationURL":"","tokenURL":"","userInfoURL":"","prompt":[],"maxAge":null}`), &m)

		if !m.MaxAge.IsNull() {
			t.Errorf("max_age = %s, want null", m.MaxAge)
		}
		if m.Prompt.IsNull() || m.Prompt.IsUnknown() || len(m.Prompt.Elements()) != 0 {
			t.Errorf("prompt = %s, want an empty set", m.Prompt)
		}
		if !m.AuthorizationURL.IsNull() {
			t.Errorf("authorization_url = %s, want null", m.AuthorizationURL)
		}

		applyOAuthResponse(p, decodeBody(t, `{"$id":"oidc","enabled":true,"clientId":"abc","clientSecret":"","wellKnownURL":"","prompt":["login"],"maxAge":600}`), &m)
		if m.MaxAge.ValueInt64() != 600 {
			t.Errorf("max_age = %s, want 600", m.MaxAge)
		}
	})

	t.Run("arguments of other providers are null", func(t *testing.T) {
		t.Parallel()
		p, _ := lookupOAuthProvider("amazon")
		m := emptyModel(p)
		applyOAuthResponse(p, decodeBody(t, `{"$id":"amazon","enabled":false,"clientId":"","clientSecret":""}`), &m)
		for name := range oauthExtras {
			if isSet(m.extra(name)) {
				t.Errorf("%s is set for amazon", name)
			}
			switch v := m.extra(name).(type) {
			case *types.Set:
				if v.IsUnknown() {
					t.Errorf("%s left unknown", name)
				}
			case *types.Int64:
				if v.IsUnknown() {
					t.Errorf("%s left unknown", name)
				}
			}
		}
	})
}

func TestValidateOAuthProviderConfig(t *testing.T) {
	t.Parallel()

	github, _ := lookupOAuthProvider("github")
	config := emptyModel(github)
	config.Endpoint = types.StringValue("example.com")
	config.MaxAge = types.Int64Value(60)
	config.Prompt = stringSet("consent")
	diags := validateOAuthProviderConfig(&config)
	if diags.ErrorsCount() != 2 {
		t.Fatalf("got %d errors, want 2 (endpoint and max_age): %v", diags.ErrorsCount(), diags)
	}
	for _, d := range diags.Errors() {
		if !strings.Contains(d.Detail(), "github") {
			t.Errorf("error does not name the provider: %s", d.Detail())
		}
	}
	if !strings.Contains(diags.Errors()[0].Detail(), "auth0, authentik, fusionauth, gitlab, keycloak") {
		t.Errorf("endpoint error does not list the providers that take it: %s", diags.Errors()[0].Detail())
	}

	// A value not known until apply may turn out null.
	config = emptyModel(github)
	config.Endpoint = types.StringUnknown()
	if diags := validateOAuthProviderConfig(&config); diags.HasError() {
		t.Errorf("unknown value rejected: %v", diags)
	}

	for _, p := range oauthProviders {
		config := fullModel(p)
		if diags := validateOAuthProviderConfig(&config); diags.HasError() {
			t.Errorf("%s rejects its own arguments: %v", p.id, diags)
		}
	}
}

// Every attribute named in the table must exist, and every provider-specific
// attribute must be taken by some provider, or it is dead schema.
func TestOAuthExtrasAreConsistent(t *testing.T) {
	t.Parallel()

	used := map[string]bool{}
	seen := map[string]bool{}
	for _, p := range oauthProviders {
		if seen[p.id] {
			t.Errorf("provider %s listed twice", p.id)
		}
		seen[p.id] = true
		for _, name := range p.extras {
			if _, ok := oauthExtras[name]; !ok {
				t.Errorf("%s takes %s, which oauthExtras does not define", p.id, name)
			}
			used[name] = true
		}
	}
	m := oauthProviderModel{}
	for name := range oauthExtras {
		if !used[name] {
			t.Errorf("no provider takes %s", name)
		}
		_ = m.extra(name) // panics on an attribute without a field
	}
}
