---
page_title: "API key scopes"
subcategory: "Guides"
description: |-
  Which credential and which scopes each resource needs, and how to read the errors when one is wrong.
---

# API key scopes

Most first failures with this provider are credential failures, and the error
Appwrite returns for them is the same regardless of which of several different
mistakes you made. This guide covers how to tell them apart.

## Two credentials, not one

The provider takes two, because Appwrite has two kinds of administrative route:

- `api_key` — a **project** key. Used for everything inside a project: tables,
  buckets, users, functions, sites, webhooks.
- `organization_api_key` — an **organization** key. Used for organization-level
  administration: creating projects, and managing a project's API keys.

If `organization_api_key` is not set it falls back to `api_key`. That fallback
exists for backwards compatibility and is not a good configuration to adopt: it
means every project-scoped operation runs with a credential broad enough to
create projects. Set both explicitly.

## Key types are detectable, and the provider checks them

Modern Appwrite keys carry their type as a prefix — `standard_`, `ephemeral_`,
`organization_`, `account_`, `oauth2_`. The provider reads that prefix and
rejects a credential that cannot possibly work before making the request, so
using an organization key as `api_key` produces an error naming the problem
rather than a generic 401 from the server.

Keys without a prefix are legacy, and the provider passes them through for the
server to judge. If you are using one, you will get the server's error rather
than the provider's clearer one; that is a reason to rotate it.

## Scopes per resource

A key with the wrong scopes returns `general_unauthorized_scope`. The provider
appends the scopes the resource needs to that error, so the message tells you
what to add. For reference:

| Resource | Credential | Scopes |
| --- | --- | --- |
| `appwrite_project` | organization | `projects.read`, `projects.write` |
| `appwrite_project_key` | organization | `keys.read`, `keys.write` |
| `appwrite_project_ephemeral_key` | organization | `keys.write` |
| `appwrite_proxy_rule` | project | `rules.read`, `rules.write` |
| `appwrite_auth_jwt` | project | `users.read`, `sessions.write` |
| `appwrite_auth_session` | project | `sessions.write` |

## The DocumentsDB and VectorsDB trap

DocumentsDB and VectorsDB resources are gated behind the **legacy Databases**
scopes, not the TablesDB ones. They need `databases.write`,
`collections.write` and `documents.write` — not `tables.write` or `rows.write`,
despite the products being newer than either.

Worse, the Appwrite Console does not always render checkboxes for those scopes.
When it does not, set them through the project keys API directly: a key's scope
list accepts names the Console will not show you.

The provider says this in the error message when one of these resources fails on
scopes, because there is no way to work it out from the resource name.

## Debugging a credential error

Set `TF_LOG=DEBUG` and look for the request that failed. The provider logs every
request and response, with keys, JWTs and session cookies redacted. See the
[debugging guide](debugging.md) for what the output contains.
