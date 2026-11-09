package project

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/appwrite/sdk-for-go/v7/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type oauthProviderModel struct {
	ID                    types.String `tfsdk:"id"`
	ProviderID            types.String `tfsdk:"provider_id"`
	Enabled               types.Bool   `tfsdk:"enabled"`
	ClientID              types.String `tfsdk:"client_id"`
	ClientSecret          types.String `tfsdk:"client_secret"`
	ClientSecretWO        types.String `tfsdk:"client_secret_wo"`
	ClientSecretWOVersion types.Int64  `tfsdk:"client_secret_wo_version"`
	Endpoint              types.String `tfsdk:"endpoint"`
	RealmName             types.String `tfsdk:"realm_name"`
	Tenant                types.String `tfsdk:"tenant"`
	Domain                types.String `tfsdk:"domain"`
	AuthorizationServerID types.String `tfsdk:"authorization_server_id"`
	WellKnownURL          types.String `tfsdk:"well_known_url"`
	AuthorizationURL      types.String `tfsdk:"authorization_url"`
	TokenURL              types.String `tfsdk:"token_url"`
	UserInfoURL           types.String `tfsdk:"user_info_url"`
	MaxAge                types.Int64  `tfsdk:"max_age"`
	Prompt                types.Set    `tfsdk:"prompt"`
	NativeClientIDs       types.Set    `tfsdk:"native_client_ids"`
	NativeEnabled         types.Bool   `tfsdk:"native_enabled"`
	KeyID                 types.String `tfsdk:"key_id"`
	TeamID                types.String `tfsdk:"team_id"`
	ProjectID             types.String `tfsdk:"project_id"`
}

// extra returns a pointer to the field holding a provider-specific attribute,
// so the request and response mapping can work from oauthExtras rather than
// naming every field.
func (m *oauthProviderModel) extra(attribute string) any {
	switch attribute {
	case attrEndpoint:
		return &m.Endpoint
	case attrRealmName:
		return &m.RealmName
	case attrTenant:
		return &m.Tenant
	case attrDomain:
		return &m.Domain
	case attrAuthorizationServerID:
		return &m.AuthorizationServerID
	case attrWellKnownURL:
		return &m.WellKnownURL
	case attrAuthorizationURL:
		return &m.AuthorizationURL
	case attrTokenURL:
		return &m.TokenURL
	case attrUserInfoURL:
		return &m.UserInfoURL
	case attrMaxAge:
		return &m.MaxAge
	case attrPrompt:
		return &m.Prompt
	case attrNativeClientIDs:
		return &m.NativeClientIDs
	case attrNativeEnabled:
		return &m.NativeEnabled
	case attrKeyID:
		return &m.KeyID
	case attrTeamID:
		return &m.TeamID
	}
	panic(fmt.Sprintf("oauthProviderModel has no field for %q", attribute))
}

// updateParams builds the request that makes a provider match the plan.
//
// Every argument the provider takes is sent, so one left out of the
// configuration is cleared rather than left as it was: Appwrite keeps the
// stored value of any parameter a request omits. The exceptions are the
// secret, which is sent only when the configuration has one, and the settings
// Appwrite cannot clear (see oauthExtra.serverDefault), which are sent only
// when set.
//
// enabled is always sent. A request that omits it, and leaves a client ID and
// secret behind, makes Appwrite try to enable the provider by itself.
func updateParams(p oauthProvider, plan *oauthProviderModel, secret string, hasSecret bool) map[string]any {
	params := map[string]any{
		"enabled":         plan.Enabled.ValueBool(),
		p.clientIDParam(): plan.ClientID.ValueString(),
	}
	if hasSecret {
		params[p.secretParam()] = secret
	}
	for _, name := range p.extras {
		extra := oauthExtras[name]
		switch v := plan.extra(name).(type) {
		case *types.String:
			params[extra.key] = v.ValueString()
		case *types.Bool:
			params[extra.key] = v.ValueBool()
		case *types.Int64:
			if v.IsNull() || v.IsUnknown() {
				continue
			}
			params[extra.key] = v.ValueInt64()
		case *types.Set:
			if v.IsNull() || v.IsUnknown() {
				if extra.serverDefault {
					continue
				}
				params[extra.key] = []string{}
				continue
			}
			params[extra.key] = setStrings(*v)
		}
	}
	return params
}

// resetParams builds the request that destroy sends: the provider disabled,
// its credentials and every other setting Appwrite can clear emptied. Settings
// Appwrite cannot clear are left out.
func resetParams(p oauthProvider) map[string]any {
	params := map[string]any{
		"enabled":         false,
		p.clientIDParam(): "",
		p.secretParam():   "",
	}
	for _, name := range p.extras {
		extra := oauthExtras[name]
		if extra.serverDefault {
			continue
		}
		switch extra.kind {
		case extraString:
			params[extra.key] = ""
		case extraBool:
			params[extra.key] = false
		case extraSet:
			params[extra.key] = []string{}
		case extraInt64:
			// No integer setting can be cleared; see serverDefault.
		}
	}
	return params
}

// applyOAuthResponse copies a provider read from Appwrite into the model.
//
// The secret is never returned, so client_secret keeps whatever the model
// holds. An argument returned empty that the model has as null stays null, so
// a provider whose configuration leaves it unset does not plan to set it to
// the empty string. A key missing from the response keeps the model's value:
// the SDK takes some settings that older Appwrite versions accept and ignore,
// and those would otherwise show as a diff on every plan. Attributes the
// provider does not take are null.
func applyOAuthResponse(p oauthProvider, body map[string]any, m *oauthProviderModel) {
	m.ID = types.StringValue(p.id)
	m.ProviderID = types.StringValue(p.id)
	enabled, _ := body["enabled"].(bool)
	m.Enabled = types.BoolValue(enabled)
	readString(&m.ClientID, body, p.clientIDParam(), false)

	for name, extra := range oauthExtras {
		field := m.extra(name)
		if !p.takes(name) {
			clearField(field)
			continue
		}
		switch v := field.(type) {
		case *types.String:
			readString(v, body, extra.key, extra.notEchoed)
		case *types.Bool:
			readBool(v, body, extra.key)
		case *types.Int64:
			readInt64(v, body, extra.key)
		case *types.Set:
			readSet(v, body, extra.key, extra.serverDefault)
		}
	}
}

func clearField(field any) {
	switch v := field.(type) {
	case *types.String:
		*v = types.StringNull()
	case *types.Bool:
		*v = types.BoolNull()
	case *types.Int64:
		*v = types.Int64Null()
	case *types.Set:
		*v = types.SetNull(types.StringType)
	}
}

func readString(field *types.String, body map[string]any, key string, notEchoed bool) {
	raw, present := body[key]
	value, _ := raw.(string)
	if !present || (value == "" && (notEchoed || field.IsNull() || field.IsUnknown())) {
		if field.IsUnknown() {
			*field = types.StringNull()
		}
		return
	}
	*field = types.StringValue(value)
}

func readBool(field *types.Bool, body map[string]any, key string) {
	raw, present := body[key]
	value, _ := raw.(bool)
	if !present || (!value && (field.IsNull() || field.IsUnknown())) {
		if field.IsUnknown() {
			*field = types.BoolNull()
		}
		return
	}
	*field = types.BoolValue(value)
}

func readInt64(field *types.Int64, body map[string]any, key string) {
	raw, present := body[key]
	if !present {
		if field.IsUnknown() {
			*field = types.Int64Null()
		}
		return
	}
	number, ok := raw.(json.Number)
	if !ok {
		*field = types.Int64Null()
		return
	}
	value, err := number.Int64()
	if err != nil {
		*field = types.Int64Null()
		return
	}
	*field = types.Int64Value(value)
}

func readSet(field *types.Set, body map[string]any, key string, serverDefault bool) {
	raw, present := body[key]
	if !present {
		if field.IsUnknown() {
			*field = types.SetNull(types.StringType)
		}
		return
	}
	items, _ := raw.([]any)
	if len(items) == 0 && !serverDefault && (field.IsNull() || field.IsUnknown()) {
		*field = types.SetNull(types.StringType)
		return
	}
	elements := make([]attr.Value, 0, len(items))
	for _, item := range items {
		s, _ := item.(string)
		elements = append(elements, types.StringValue(s))
	}
	*field = types.SetValueMust(types.StringType, elements)
}

func setStrings(set types.Set) []string {
	values := make([]string, 0, len(set.Elements()))
	for _, element := range set.Elements() {
		if s, ok := element.(types.String); ok {
			values = append(values, s.ValueString())
		}
	}
	sort.Strings(values)
	return values
}

var (
	oauthProviderLocksMu sync.Mutex
	// oauthProviderLocks holds one mutex per project ID.
	oauthProviderLocks = map[string]*sync.Mutex{}
)

// lockOAuthProviders serializes OAuth2 provider updates within a project.
//
// Appwrite keeps every provider of a project in one attribute of the project
// document, and an update writes that whole attribute back from the copy it
// read when the request started. Two updates in flight at once therefore lose
// all but one of them: ten concurrent updates to ten different providers left
// one applied on Appwrite 2.2.0. Terraform applies independent resources in
// parallel, so a configuration with several providers hits this on its first
// apply. Holding the lock until the response arrives makes them sequential.
func lockOAuthProviders(projectID string) func() {
	oauthProviderLocksMu.Lock()
	mu, ok := oauthProviderLocks[projectID]
	if !ok {
		mu = &sync.Mutex{}
		oauthProviderLocks[projectID] = mu
	}
	oauthProviderLocksMu.Unlock()

	mu.Lock()
	return mu.Unlock
}

func oauthProviderPath(id string) string {
	return "/project/oauth2/" + client.EncodePath(id)
}

// getOAuthProvider reads a provider's settings.
//
// The SDK's GetOAuth2Provider returns a different model type per provider,
// and fails with "unable to match response" for appwrite, paypal and
// tradeshift, whose IDs it does not list. The response is decoded generically
// instead and mapped through the provider table.
func getOAuthProvider(c client.Client, id string) (map[string]any, error) {
	return callOAuthProvider(c, "GET", id, map[string]any{})
}

// updateOAuthProvider sends a provider's update route and returns the
// settings Appwrite stored, in the same shape getOAuthProvider reads. It holds
// the project's lock for the duration; see lockOAuthProviders.
func updateOAuthProvider(c client.Client, id string, params map[string]any) (map[string]any, error) {
	unlock := lockOAuthProviders(c.Config["project"])
	defer unlock()
	return callOAuthProvider(c, "PATCH", id, params)
}

func callOAuthProvider(c client.Client, method, id string, params map[string]any) (map[string]any, error) {
	headers := map[string]any{
		// client.Call does not add the project header; see common.GetColumnRawJSON.
		"X-Appwrite-Project": c.Config["project"],
		"accept":             contentTypeJSON,
	}
	if method != "GET" {
		headers["content-type"] = contentTypeJSON
	}
	resp, err := c.Call(method, oauthProviderPath(id), headers, params)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(resp.Type, contentTypeJSON) {
		return nil, fmt.Errorf("unexpected response type: %s", resp.Type)
	}
	raw, err := client.ResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("reading OAuth2 provider response: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	body := map[string]any{}
	if err := decoder.Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding OAuth2 provider response: %w", err)
	}
	return body, nil
}
