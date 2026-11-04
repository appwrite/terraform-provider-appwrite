---
page_title: "Resource: appwrite_waf_rule"
subcategory: "WAF"
description: |-
  Manages a web application firewall (WAF) rule on an Appwrite Cloud project.
---

# Resource: appwrite_waf_rule

Manages a web application firewall (WAF) rule on an Appwrite Cloud project.

The web application firewall is available on Appwrite Cloud only; self-hosted Appwrite does not serve its API. A standard project API key with `wafRules.read` and `wafRules.write` scopes is required. Configure it with the provider's `api_key` attribute or `APPWRITE_API_KEY`. Appwrite Cloud limits how many rules a project can hold, depending on its plan.

## How rules are evaluated

Appwrite evaluates a project's enabled rules in ascending `priority` order, and the first rule whose conditions all match decides what happens to the request:

- `bypass` lets the request through without evaluating the rules after it.
- `deny` rejects the request.
- `challenge` makes the visitor solve a proof-of-work challenge, then lets them through for `ttl` seconds.
- `rateLimit` counts the request against `limit` per `interval`, keyed by client IP or signed-in user, and rejects it once the limit is reached.
- `redirect` responds with `status_code` and a redirect to `location`.

Give rules that can match the same request different priorities: the order of rules with equal priority is not defined.

`resource_type = "api"`, the default, applies a rule to the project's API. Rules apply to requests made with an API key too, so a broad `deny` or `rateLimit` rule can affect Terraform's own requests for other resources in the project. Requests that manage WAF rules are not filtered, so a rule can always be changed or removed. `functions` and `sites` scope a rule to one function or site, named by `resource_id`.

## Conditions

Each entry in `conditions` is one condition as a JSON object, most easily written with `jsonencode`. A condition has a `method`, the request `attribute` it tests, and the `values` to test against. A request matches a rule when it matches every condition; a rule without conditions matches every request.

```hcl
conditions = [
  jsonencode({ method = "equal", attribute = "ip", values = ["203.0.113.0/24"] }),
  jsonencode({ method = "startsWith", attribute = "path", values = ["/v1/functions"] }),
]
```

Methods include `equal`, `notEqual`, `lessThan`, `lessThanEqual`, `greaterThan`, `greaterThanEqual`, `between`, `notBetween`, `contains`, `notContains`, `startsWith`, `notStartsWith`, `endsWith`, `notEndsWith`, `isNull` and `isNotNull`. `and` and `or` take no attribute and nest further conditions in `values`:

```hcl
jsonencode({
  method = "or"
  values = [
    { method = "equal", attribute = "country", values = ["NL"] },
    { method = "equal", attribute = "country", values = ["BE"] },
  ]
})
```

Attributes include `ip` (an address or a CIDR block), `method`, `path`, `host`, `country`, `continent`, `headers.<name>`, `query.<key>`, `queryKeys`, `userAgent`, `os`, `osVersion`, `browser` and `browserVersion`. Conditions on `city` and `state` require the premium Geo DB add-on.

The provider checks the structure of each condition at plan time and leaves the method and attribute names to Appwrite, which rejects unknown ones when the rule is created or updated. A rule holds up to 100 conditions, counting nested ones.

Appwrite stores conditions in its own form, keeping only `method`, `attribute` and `values`. The provider compares conditions in that form, so spacing and key order in the configuration are not shown as changes.

## Arguments by rule type

| `type` | Required | Optional |
| --- | --- | --- |
| `bypass`, `deny` | | |
| `challenge` | | `challenge_type`, `difficulty`, `ttl` |
| `rateLimit` | `limit`, `interval` | `key`, `strategy`, `max_bucket_size` (with `strategy = "tokenBucket"` only) |
| `redirect` | `location`, `status_code` | |

The arguments of one type are rejected for the others. An optional argument that is left out is planned with its default, and removing one from the configuration sets it back to its default.

Changing `id`, `type`, `strategy` or `project_id` replaces the rule. Removing every condition from a rule that has some also replaces it, because Appwrite cannot clear a rule's conditions in place. Every other argument is updated in place, including `resource_type` and `resource_id`.

Rule IDs are unique across Appwrite Cloud, not only within a project. Leave `id` unset to have one generated.

## Example Usage

```terraform
# Let the office network through before any other rule is evaluated.
resource "appwrite_waf_rule" "office" {
  type     = "bypass"
  name     = "Office network"
  priority = -100
  conditions = [
    jsonencode({ method = "equal", attribute = "ip", values = ["198.51.100.0/24"] }),
  ]
}

resource "appwrite_waf_rule" "block_scrapers" {
  id          = "block-scrapers"
  type        = "deny"
  name        = "Block scrapers"
  description = "Known scraper user agents"
  conditions = [
    jsonencode({ method = "contains", attribute = "userAgent", values = ["python-requests", "scrapy"] }),
  ]
}

# Every condition must match. Nest conditions in "or" to match any of them.
resource "appwrite_waf_rule" "sign_in_limit" {
  type     = "rateLimit"
  name     = "Sign-in attempts"
  limit    = 10
  interval = 60
  strategy = "slidingWindow"
  conditions = [
    jsonencode({ method = "equal", attribute = "method", values = ["POST"] }),
    jsonencode({
      method = "or"
      values = [
        { method = "startsWith", attribute = "path", values = ["/v1/account/sessions"] },
        { method = "startsWith", attribute = "path", values = ["/v1/account/tokens"] },
      ]
    }),
  ]
}

resource "appwrite_waf_rule" "challenge_outside_eu" {
  type       = "challenge"
  name       = "Challenge traffic from outside Europe"
  difficulty = 2
  conditions = [
    jsonencode({ method = "notEqual", attribute = "continent", values = ["EU"] }),
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The rule name, up to 128 characters.
- `type` (String) What the rule does with a request it matches: bypass, challenge, deny, rateLimit, redirect. Changing this forces a new resource to be created.

### Optional

- `challenge_type` (String) The challenge a matching visitor must pass. Only compute, a proof-of-work challenge, is available. Defaults to compute for challenge rules; not accepted for other types.
- `conditions` (List of String) The conditions a request must all meet for the rule to match, each a JSON object usually written with jsonencode, such as jsonencode({ method = "equal", attribute = "ip", values = ["198.51.100.10"] }). An and or an or condition nests further conditions in values. A rule with no conditions matches every request. Up to 100 conditions, counting nested ones. Removing every condition from a rule replaces it, because Appwrite cannot clear a rule's conditions in place.
- `description` (String) A description of the rule, up to 1024 characters.
- `difficulty` (Number) The challenge difficulty, from 1 (easiest) to 5 (hardest). Defaults to 3 for challenge rules; not accepted for other types.
- `enabled` (Boolean) Whether the rule is enforced. Defaults to true.
- `id` (String) The rule ID. Generated when not set. Rule IDs are unique across Appwrite Cloud, not only within the project. Changing this forces a new resource to be created.
- `interval` (Number) The rate limit interval in seconds, from 1 to 86400. Required for rateLimit rules; not accepted for other types.
- `key` (String) What requests are counted by: ip counts per client IP, userId per signed-in user. A userId rule does not apply to requests without a signed-in user. Defaults to ip for rateLimit rules; not accepted for other types.
- `limit` (Number) The number of matching requests allowed per interval, from 1 to 1000000. Required for rateLimit rules; not accepted for other types.
- `location` (String) Where a redirect rule sends matching requests, as a path or URL of up to 2048 characters. Required for redirect rules; not accepted for other types.
- `max_bucket_size` (Number) The largest burst a tokenBucket rule allows, in requests, from 1 to 1000000. Defaults to limit. Accepted only when strategy is tokenBucket.
- `priority` (Number) The evaluation order, from -100000 to 100000. Rules with lower values are evaluated first, and the first rule that matches a request decides it. Defaults to 0.
- `project_id` (String) The Appwrite project ID. Defaults to the provider-level project_id. Changing this forces a new resource to be created.
- `resource_id` (String) The ID of the function or site the rule applies to. Required when resource_type is functions or sites, and not accepted for api.
- `resource_type` (String) What the rule applies to: api, functions, sites. api covers the whole project API. Defaults to api.
- `status_code` (Number) The HTTP status a redirect rule responds with, from 300 to 399. Required for redirect rules; not accepted for other types.
- `strategy` (String) How requests are counted: fixedWindow counts per interval, slidingWindow weights in the previous interval for smoother limiting, and tokenBucket refills continuously at limit per interval to allow short bursts. Defaults to fixedWindow for rateLimit rules; not accepted for other types. Changing this forces a new resource to be created.
- `ttl` (Number) How long, in seconds, a visitor who passed the challenge is let through before being challenged again, from 900 to 86400. Defaults to 1800 for challenge rules; not accepted for other types.

### Read-Only

- `created_at` (String) The rule creation timestamp in ISO 8601 format.
- `updated_at` (String) The rule last update timestamp in ISO 8601 format.

## Import

Using `terraform import` with a project and rule ID:

```shell
terraform import appwrite_waf_rule.example project-id/block-scrapers
```

When `project_id` is configured on the provider, the rule ID alone can be used:

```shell
terraform import appwrite_waf_rule.example block-scrapers
```

Using an import block (Terraform v1.5.0+):

```hcl
import {
  to = appwrite_waf_rule.example
  id = "project-id/block-scrapers"
}
```

An import stores each condition the way `jsonencode` writes it, so a configuration that uses `jsonencode` plans no change for them. A condition written as a literal JSON string in another key order or spacing shows as an in-place update on the first plan after import. Applying it sends the same conditions and changes nothing on the server.

## See Also

- [appwrite_proxy_rule](proxy_rule.md) - Serve a site or function on a custom domain
- [appwrite_domain](domain.md) - Host a domain's DNS on Appwrite
