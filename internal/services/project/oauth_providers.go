package project

import (
	"fmt"
	"sort"
	"strings"
)

// oauthProvider describes one OAuth2 provider's update route.
//
// Every provider has its own route, /project/oauth2/<id>, and they differ only
// in what they call their two credentials and which extra settings they take.
// That is all this table records, so adding a provider is one line rather than
// one more branch in every CRUD method. TestOAuthProvidersMatchSDK checks each
// entry against the SDK's UpdateOAuth2<label> method.
type oauthProvider struct {
	// id is the provider ID in the route and in the response's $id.
	id string
	// label is the suffix of the SDK's UpdateOAuth2<label> method.
	label string
	// clientIDKey and secretKey are the request and response keys that
	// client_id and client_secret map to. Empty means clientId and
	// clientSecret, which most providers use.
	clientIDKey string
	secretKey   string
	// extras are the provider-specific attributes it accepts, by attribute
	// name. See oauthExtras.
	extras []string
}

func (p oauthProvider) clientIDParam() string {
	if p.clientIDKey != "" {
		return p.clientIDKey
	}
	return "clientId"
}

func (p oauthProvider) secretParam() string {
	if p.secretKey != "" {
		return p.secretKey
	}
	return "clientSecret"
}

func (p oauthProvider) takes(attribute string) bool {
	for _, e := range p.extras {
		if e == attribute {
			return true
		}
	}
	return false
}

// Provider-specific attribute names.
const (
	attrEndpoint              = "endpoint"
	attrRealmName             = "realm_name"
	attrTenant                = "tenant"
	attrDomain                = "domain"
	attrAuthorizationServerID = "authorization_server_id"
	attrWellKnownURL          = "well_known_url"
	attrAuthorizationURL      = "authorization_url"
	attrTokenURL              = "token_url"
	attrUserInfoURL           = "user_info_url"
	attrMaxAge                = "max_age"
	attrPrompt                = "prompt"
	attrNativeClientIDs       = "native_client_ids"
	attrNativeEnabled         = "native_enabled"
	attrKeyID                 = "key_id"
	attrTeamID                = "team_id"
)

// keySecretKey is the secret's request key for several providers.
const keySecretKey = "secretKey"

// oauthProviders is every provider the SDK has an update route for.
var oauthProviders = []oauthProvider{
	{id: "amazon", label: "Amazon"},
	{id: "apple", label: "Apple", clientIDKey: "serviceId", secretKey: "p8File", extras: []string{attrKeyID, attrTeamID, attrNativeClientIDs, attrNativeEnabled}},
	{id: "appwrite", label: "Appwrite"},
	{id: "auth0", label: "Auth0", extras: []string{attrEndpoint, attrPrompt}},
	{id: "authentik", label: "Authentik", extras: []string{attrEndpoint}},
	{id: "autodesk", label: "Autodesk"},
	{id: "bitbucket", label: "Bitbucket", clientIDKey: "key", secretKey: "secret"},
	{id: "bitly", label: "Bitly"},
	{id: "box", label: "Box"},
	{id: "cloudflare", label: "Cloudflare"},
	{id: "dailymotion", label: "Dailymotion", clientIDKey: "apiKey", secretKey: "apiSecret"},
	{id: "discord", label: "Discord", extras: []string{attrPrompt}},
	{id: "disqus", label: "Disqus", clientIDKey: "publicKey", secretKey: keySecretKey},
	{id: "dropbox", label: "Dropbox", clientIDKey: "appKey", secretKey: "appSecret"},
	{id: "etsy", label: "Etsy", clientIDKey: "keyString", secretKey: "sharedSecret"},
	{id: "facebook", label: "Facebook", clientIDKey: "appId", secretKey: "appSecret"},
	{id: "figma", label: "Figma"},
	{id: "fusionauth", label: "FusionAuth", extras: []string{attrEndpoint}},
	{id: "github", label: "GitHub", extras: []string{attrPrompt}},
	{id: "gitlab", label: "Gitlab", clientIDKey: "applicationId", secretKey: "secret", extras: []string{attrEndpoint}},
	{id: "google", label: "Google", extras: []string{attrPrompt, attrNativeClientIDs, attrNativeEnabled}},
	{id: "huggingface", label: "HuggingFace"},
	{id: "kakao", label: "Kakao", extras: []string{attrPrompt}},
	{id: "keycloak", label: "Keycloak", extras: []string{attrEndpoint, attrRealmName}},
	{id: "kick", label: "Kick"},
	{id: "linkedin", label: "Linkedin", secretKey: "primaryClientSecret"},
	{id: "microsoft", label: "Microsoft", clientIDKey: "applicationId", secretKey: "applicationSecret", extras: []string{attrTenant, attrPrompt}},
	{id: "notion", label: "Notion", clientIDKey: "oauthClientId", secretKey: "oauthClientSecret"},
	{id: "oidc", label: "Oidc", extras: []string{attrWellKnownURL, attrAuthorizationURL, attrTokenURL, attrUserInfoURL, attrPrompt, attrMaxAge}},
	{id: "okta", label: "Okta", extras: []string{attrDomain, attrAuthorizationServerID, attrPrompt}},
	{id: "paypal", label: "Paypal", secretKey: keySecretKey},
	{id: "paypalSandbox", label: "PaypalSandbox", secretKey: keySecretKey},
	{id: "podio", label: "Podio"},
	{id: "resend", label: "Resend"},
	{id: "salesforce", label: "Salesforce", clientIDKey: "customerKey", secretKey: "customerSecret", extras: []string{attrPrompt}},
	{id: "slack", label: "Slack"},
	{id: "spotify", label: "Spotify"},
	{id: "stripe", label: "Stripe", secretKey: "apiSecretKey"},
	{id: "tiktok", label: "TikTok"},
	{id: "tradeshift", label: "Tradeshift", clientIDKey: "oauth2ClientId", secretKey: "oauth2ClientSecret"},
	// The SDK names this one TradeshiftSandbox, but its route and ID are
	// tradeshiftBox.
	{id: "tradeshiftBox", label: "TradeshiftSandbox", clientIDKey: "oauth2ClientId", secretKey: "oauth2ClientSecret"},
	{id: "twitch", label: "Twitch"},
	{id: "vipps", label: "Vipps"},
	{id: "vippsSandbox", label: "VippsSandbox"},
	{id: "webflow", label: "Webflow"},
	{id: "wordpress", label: "WordPress"},
	{id: "x", label: "X", clientIDKey: "customerKey", secretKey: keySecretKey},
	{id: "yahoo", label: "Yahoo"},
	{id: "yandex", label: "Yandex"},
	{id: "zoho", label: "Zoho", extras: []string{attrPrompt}},
	{id: "zoom", label: "Zoom"},
}

func lookupOAuthProvider(id string) (oauthProvider, bool) {
	for _, p := range oauthProviders {
		if p.id == id {
			return p, true
		}
	}
	return oauthProvider{}, false
}

func oauthProviderIDs() []string {
	ids := make([]string, 0, len(oauthProviders))
	for _, p := range oauthProviders {
		ids = append(ids, p.id)
	}
	return ids
}

type extraKind int

const (
	extraString extraKind = iota
	extraBool
	extraSet
	extraInt64
)

// oauthExtra is a provider-specific setting.
type oauthExtra struct {
	// key is the request and response key. It is the same for every
	// provider that takes the setting.
	key         string
	kind        extraKind
	description string
	// serverDefault marks a setting Appwrite fills in when it is not given
	// and cannot be cleared again: Google rejects an empty prompt, and an
	// OIDC max_age once set can only be changed, not removed. Such a setting
	// is Optional and Computed, is left out of the request when the
	// configuration does not set it, and is left as it is on destroy.
	serverDefault bool
	// notEchoed marks a setting some Appwrite versions return and others
	// always return empty. An empty value read back is not taken to mean it
	// was cleared.
	notEchoed bool
}

var oauthExtras = map[string]oauthExtra{
	attrEndpoint: {
		key:  "endpoint",
		kind: extraString,
		description: "The provider instance to sign in with: the domain of an Auth0, Authentik or FusionAuth tenant, " +
			"the Keycloak server's domain, or the URL of a self-hosted GitLab instance.",
	},
	attrRealmName: {
		key:         "realmName",
		kind:        extraString,
		description: "The Keycloak realm name.",
	},
	attrTenant: {
		key:  "tenant",
		kind: extraString,
		description: "The Microsoft Entra ID tenant to accept accounts from: common, organizations, consumers or a " +
			"tenant ID.",
	},
	attrDomain: {
		key:         "domain",
		kind:        extraString,
		description: "The Okta organization domain, for example trial-6400025.okta.com. Required to enable Okta.",
	},
	attrAuthorizationServerID: {
		key:         "authorizationServerId",
		kind:        extraString,
		description: "The ID of a custom Okta authorization server. Leave unset to use the organization's default.",
	},
	attrWellKnownURL: {
		key:  "wellKnownURL",
		kind: extraString,
		description: "The OpenID Connect discovery document URL. Enabling OIDC needs either this or all of " +
			"authorization_url, token_url and user_info_url.",
	},
	attrAuthorizationURL: {
		key:         "authorizationURL",
		kind:        extraString,
		description: "The OpenID Connect authorization endpoint URL.",
	},
	attrTokenURL: {
		key:         "tokenURL",
		kind:        extraString,
		description: "The OpenID Connect token endpoint URL.",
	},
	attrUserInfoURL: {
		key:         "userInfoURL",
		kind:        extraString,
		description: "The OpenID Connect user info endpoint URL.",
	},
	attrMaxAge: {
		key:  "maxAge",
		kind: extraInt64,
		description: "For OIDC, the maximum time in seconds since the user last authenticated with the provider " +
			"before they must sign in again. Appwrite cannot unset it once set, so removing it from the " +
			"configuration keeps the last value.",
		serverDefault: true,
	},
	attrPrompt: {
		key:  "prompt",
		kind: extraSet,
		description: "The prompt values sent with the authorization request, such as consent, login, " +
			"select_account or none. none must be the only value. Appwrite fills in its default when unset " +
			"(consent for Google), and removing it from the configuration keeps the last value. Appwrite " +
			"2.2 accepts prompt only for google and oidc and ignores it for other providers.",
		serverDefault: true,
	},
	attrNativeClientIDs: {
		key:  "nativeClientIds",
		kind: extraSet,
		description: "Additional client IDs (Google) or app bundle IDs (Apple) accepted as the audience of ID " +
			"tokens for native sign-in.",
	},
	attrNativeEnabled: {
		key:  "nativeEnabled",
		kind: extraBool,
		description: "Whether sessions can be created from ID tokens obtained on the device by native Google or " +
			"Apple sign-in. Independent of enabled, and needs client_id or native_client_ids.",
	},
	attrKeyID: {
		key:         "keyId",
		kind:        extraString,
		description: "The Sign in with Apple key ID.",
		notEchoed:   true,
	},
	attrTeamID: {
		key:         "teamId",
		kind:        extraString,
		description: "The Apple developer team ID.",
		notEchoed:   true,
	},
}

// providersTaking lists the providers that accept an extra attribute, for its
// description and its validation error.
func providersTaking(attribute string) []string {
	var ids []string
	for _, p := range oauthProviders {
		if p.takes(attribute) {
			ids = append(ids, p.id)
		}
	}
	sort.Strings(ids)
	return ids
}

func extraDescription(attribute string) string {
	return fmt.Sprintf("%s Only for %s.", oauthExtras[attribute].description, strings.Join(providersTaking(attribute), ", "))
}

func sortedExtraNames() []string {
	names := make([]string, 0, len(oauthExtras))
	for name := range oauthExtras {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
