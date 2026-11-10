---
page_title: "Resource: appwrite_project_oauth_provider"
subcategory: "Projects"
description: |-
  Configures an OAuth2 sign-in provider of an Appwrite project. Every provider exists in every project, so creating this resource sets the provider up and destroying it disables the provider and clears its credentials.
---

# Resource: appwrite_project_oauth_provider

Configures an OAuth2 sign-in provider of an Appwrite project. Every provider exists in every project, so creating this resource sets the provider up and destroying it disables the provider and clears its credentials.

A project API key with the `project.oauth2.read` and `project.oauth2.write` scopes is required.

Every provider Appwrite supports is part of every project, enabled or not. Creating this resource therefore takes over whatever the provider is set to, from the Console or elsewhere, and destroying it disables the provider and clears its client ID, secret and other settings. Changing `provider_id` or `project_id` resets the old provider and configures the new one.

Configure each provider of a project with at most one resource. Two resources for the same provider overwrite each other on every apply.

## Credentials

`client_id` and `client_secret` are the OAuth2 application's credentials, whatever the provider calls them. For Sign in with Apple, `client_id` is the Services ID and `client_secret` is the contents of the `.p8` private key file.

Appwrite never returns the secret, so a secret changed outside Terraform is not detected, and after an import the first apply sends the configured secret. Removing the secret from the configuration leaves the stored one in place; destroying the resource clears it. Every other argument is read back, and removing one from the configuration clears it, with two exceptions: `prompt` and `max_age` fall back to Appwrite's default when unset, and removing them keeps the last value, because Appwrite cannot clear them.

Enabling some providers, for example `github`, `microsoft` and `keycloak`, makes Appwrite check the client ID and secret with the provider, and the apply fails if the provider rejects them. Set `enabled = false` to store credentials without enabling sign-in.

## Providers

Each provider takes `client_id`, `client_secret` and `enabled`. The table lists the name Appwrite's API gives the two credentials, which is usually the name the provider's own console uses, and the other arguments each provider accepts. Setting an argument a provider does not take is an error.

| `provider_id` | `client_id` in the API | `client_secret` in the API | Other arguments |
| --- | --- | --- | --- |
| `amazon` | `clientId` | `clientSecret` | |
| `apple` | `serviceId` | `p8File` | `key_id`, `team_id`, `native_client_ids`, `native_enabled` |
| `appwrite` | `clientId` | `clientSecret` | |
| `auth0` | `clientId` | `clientSecret` | `endpoint`, `prompt` |
| `authentik` | `clientId` | `clientSecret` | `endpoint` |
| `autodesk` | `clientId` | `clientSecret` | |
| `bitbucket` | `key` | `secret` | |
| `bitly` | `clientId` | `clientSecret` | |
| `box` | `clientId` | `clientSecret` | |
| `cloudflare` | `clientId` | `clientSecret` | |
| `dailymotion` | `apiKey` | `apiSecret` | |
| `discord` | `clientId` | `clientSecret` | `prompt` |
| `disqus` | `publicKey` | `secretKey` | |
| `dropbox` | `appKey` | `appSecret` | |
| `etsy` | `keyString` | `sharedSecret` | |
| `facebook` | `appId` | `appSecret` | |
| `figma` | `clientId` | `clientSecret` | |
| `fusionauth` | `clientId` | `clientSecret` | `endpoint` |
| `github` | `clientId` | `clientSecret` | `prompt` |
| `gitlab` | `applicationId` | `secret` | `endpoint` |
| `google` | `clientId` | `clientSecret` | `prompt`, `native_client_ids`, `native_enabled` |
| `huggingface` | `clientId` | `clientSecret` | |
| `kakao` | `clientId` | `clientSecret` | `prompt` |
| `keycloak` | `clientId` | `clientSecret` | `endpoint`, `realm_name` |
| `kick` | `clientId` | `clientSecret` | |
| `linkedin` | `clientId` | `primaryClientSecret` | |
| `microsoft` | `applicationId` | `applicationSecret` | `tenant`, `prompt` |
| `notion` | `oauthClientId` | `oauthClientSecret` | |
| `oidc` | `clientId` | `clientSecret` | `well_known_url`, `authorization_url`, `token_url`, `user_info_url`, `prompt`, `max_age` |
| `okta` | `clientId` | `clientSecret` | `domain`, `authorization_server_id`, `prompt` |
| `paypal` | `clientId` | `secretKey` | |
| `paypalSandbox` | `clientId` | `secretKey` | |
| `podio` | `clientId` | `clientSecret` | |
| `resend` | `clientId` | `clientSecret` | |
| `salesforce` | `customerKey` | `customerSecret` | `prompt` |
| `slack` | `clientId` | `clientSecret` | |
| `spotify` | `clientId` | `clientSecret` | |
| `stripe` | `clientId` | `apiSecretKey` | |
| `tiktok` | `clientId` | `clientSecret` | |
| `tradeshift` | `oauth2ClientId` | `oauth2ClientSecret` | |
| `tradeshiftBox` | `oauth2ClientId` | `oauth2ClientSecret` | |
| `twitch` | `clientId` | `clientSecret` | |
| `vipps` | `clientId` | `clientSecret` | |
| `vippsSandbox` | `clientId` | `clientSecret` | |
| `webflow` | `clientId` | `clientSecret` | |
| `wordpress` | `clientId` | `clientSecret` | |
| `x` | `customerKey` | `secretKey` | |
| `yahoo` | `clientId` | `clientSecret` | |
| `yandex` | `clientId` | `clientSecret` | |
| `zoho` | `clientId` | `clientSecret` | `prompt` |
| `zoom` | `clientId` | `clientSecret` | |

`webflow`, and `prompt` for providers other than `google` and `oidc`, need a newer Appwrite than 2.2. Appwrite 2.2 rejects `webflow` and ignores `prompt` for the others.

## Example Usage

```terraform
resource "appwrite_project_oauth_provider" "github" {
  provider_id              = "github"
  client_id                = "Iv1.0123456789abcdef"
  client_secret_wo         = var.github_client_secret
  client_secret_wo_version = 1
}

resource "appwrite_project_oauth_provider" "google" {
  provider_id      = "google"
  client_id        = "1234567890-abc.apps.googleusercontent.com"
  client_secret_wo = var.google_client_secret
  # Increment after rotating the secret in Google Cloud.
  client_secret_wo_version = 1
  prompt                   = ["select_account"]
}

# A generic OpenID Connect identity provider, discovered from its
# well-known configuration.
resource "appwrite_project_oauth_provider" "oidc" {
  provider_id              = "oidc"
  client_id                = "appwrite"
  client_secret_wo         = var.oidc_client_secret
  client_secret_wo_version = 1
  well_known_url           = "https://login.example.com/.well-known/openid-configuration"
}

# Sign in with Apple takes the Services ID as client_id and the contents of
# the .p8 key file as the secret.
resource "appwrite_project_oauth_provider" "apple" {
  provider_id              = "apple"
  client_id                = "com.example.web"
  client_secret_wo         = file("${path.module}/AuthKey_P4000000N8.p8")
  client_secret_wo_version = 1
  key_id                   = "P4000000N8"
  team_id                  = "D4000000R6"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `provider_id` (String) The OAuth2 provider to configure, for example github, google or oidc. The resource documentation lists every provider and the arguments it takes. Changing this forces a new resource to be created.

### Optional

> **NOTE**: [Write-only arguments](https://developer.hashicorp.com/terraform/language/resources/ephemeral#write-only-arguments) are supported in Terraform 1.11 and later.

- `authorization_server_id` (String) The ID of a custom Okta authorization server. Leave unset to use the organization's default. Only for okta.
- `authorization_url` (String) The OpenID Connect authorization endpoint URL. Only for oidc.
- `client_id` (String) The OAuth2 application's client ID, under whatever name the provider gives it: app ID, application ID, app key, or for Apple the Services ID.
- `client_secret` (String, Sensitive) The OAuth2 application's client secret, or for Apple the contents of the .p8 key file. Appwrite never returns it, so a change made outside Terraform is not detected. Stored in Terraform state; prefer client_secret_wo, which is not.
- `client_secret_wo` (String, Sensitive, [Write-only](https://developer.hashicorp.com/terraform/language/resources/ephemeral#write-only-arguments)) The client secret, as a write-only argument. Read from the configuration during apply and never persisted. Change client_secret_wo_version to apply a new value. Requires Terraform 1.11 or later.
- `client_secret_wo_version` (Number) Increment to apply a changed client_secret_wo.
- `domain` (String) The Okta organization domain, for example trial-6400025.okta.com. Required to enable Okta. Only for okta.
- `enabled` (Boolean) Whether users can sign in with the provider. Defaults to true. Enabling some providers makes Appwrite check the client ID and secret with the provider, and the apply fails if they are rejected.
- `endpoint` (String) The provider instance to sign in with: the domain of an Auth0, Authentik or FusionAuth tenant, the Keycloak server's domain, or the URL of a self-hosted GitLab instance. Only for auth0, authentik, fusionauth, gitlab, keycloak.
- `key_id` (String) The Sign in with Apple key ID. Only for apple.
- `max_age` (Number) For OIDC, the maximum time in seconds since the user last authenticated with the provider before they must sign in again. Appwrite cannot unset it once set, so removing it from the configuration keeps the last value. Only for oidc.
- `native_client_ids` (Set of String) Additional client IDs (Google) or app bundle IDs (Apple) accepted as the audience of ID tokens for native sign-in. Only for apple, google.
- `native_enabled` (Boolean) Whether sessions can be created from ID tokens obtained on the device by native Google or Apple sign-in. Independent of enabled, and needs client_id or native_client_ids. Only for apple, google.
- `project_id` (String) The Appwrite project ID. Defaults to the provider-level project_id. Changing this forces a new resource to be created.
- `prompt` (Set of String) The prompt values sent with the authorization request, such as consent, login, select_account or none. none must be the only value. Appwrite fills in its default when unset (consent for Google), and removing it from the configuration keeps the last value. Appwrite 2.2 accepts prompt only for google and oidc and ignores it for other providers. Only for auth0, discord, github, google, kakao, microsoft, oidc, okta, salesforce, zoho.
- `realm_name` (String) The Keycloak realm name. Only for keycloak.
- `team_id` (String) The Apple developer team ID. Only for apple.
- `tenant` (String) The Microsoft Entra ID tenant to accept accounts from: common, organizations, consumers or a tenant ID. Only for microsoft.
- `token_url` (String) The OpenID Connect token endpoint URL. Only for oidc.
- `user_info_url` (String) The OpenID Connect user info endpoint URL. Only for oidc.
- `well_known_url` (String) The OpenID Connect discovery document URL. Enabling OIDC needs either this or all of authorization_url, token_url and user_info_url. Only for oidc.

### Read-Only

- `id` (String) The provider ID, the same as provider_id.

## Import

Using `terraform import` with a project ID and provider ID:

```shell
terraform import appwrite_project_oauth_provider.github project-id/github
```

When `project_id` is configured on the provider, the provider ID alone can be used:

```shell
terraform import appwrite_project_oauth_provider.github github
```

Using an import block (Terraform v1.5.0+):

```hcl
import {
  to = appwrite_project_oauth_provider.github
  id = "project-id/github"
}
```

## See Also

- [appwrite_project](project.md) - Manage projects
