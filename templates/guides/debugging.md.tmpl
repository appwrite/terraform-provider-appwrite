---
page_title: "Debugging"
subcategory: "Guides"
description: |-
  How to see what the provider sent, what Appwrite replied, and which of the two is wrong.
---

# Debugging

## Turn on logging

```sh
export TF_LOG=DEBUG
terraform apply
```

Every Appwrite request and response is logged: method, URL, status, elapsed
time, and headers. To keep it out of your terminal and in a file:

```sh
export TF_LOG=DEBUG
export TF_LOG_PATH=./terraform-debug.log
terraform apply
```

`TF_LOG=TRACE` adds Terraform's own protocol traffic, which is what you want
when the suspicion is a framework or plan problem rather than an API one.

## What is redacted

API keys, JWTs, session cookies and dev keys are replaced with `(redacted)` in
the log, case-insensitively. Project and organization identifiers are not — they
are not secret and you need them to make sense of the output.

Response *bodies* are not logged. A body can contain user records, file contents
or credentials the provider has no way to classify, and a debug log is routinely
pasted into a bug report. If you need a body, get it from the server side.

This means a debug log is safe to share, with one caveat: check it anyway before
attaching it to a public issue. Redaction covers the headers the provider knows
about, and a custom reverse proxy in front of Appwrite may add its own.

## Reading a retry

A retried request appears as several request lines for the same URL, each
followed by `retrying Appwrite request` with the attempt number, the wait, and
the reason. If you see the retry budget exhausted on `429`, the fix is usually
fewer parallel operations rather than more retries:

```sh
terraform apply -parallelism=2
```

Raising `max_retries` while leaving parallelism high makes throttling more
likely, not less — every worker that hit the limit simply waits and hits it
again.

## A perpetual diff

If `terraform plan` is never empty for an unchanged configuration, the provider
is writing something to state during create that it does not read back during
refresh. Confirm it:

```sh
terraform apply
terraform plan   # should be empty
```

Then narrow it with `terraform show -json` and compare the attribute the plan
wants to change against what the API returns for the same resource. This class
of bug is what `TF_ACC_REFRESH_AFTER_APPLY` catches in the acceptance suite, and
the suite sets it for every test, so a report of one is worth filing.

## Running the provider from source

Build it and point Terraform at the binary with a development override:

```sh
make build
```

```hcl
# ~/.terraformrc
provider_installation {
  dev_overrides {
    "appwrite/appwrite" = "/path/to/terraform-provider-appwrite"
  }
  direct {}
}
```

Terraform prints a warning about the override on every command, and
`terraform init` is neither needed nor useful while one is in effect.

## Attaching a debugger

```sh
go build -gcflags="all=-N -l" -o terraform-provider-appwrite
dlv exec --headless ./terraform-provider-appwrite -- -debug
```

Delve prints a `TF_REATTACH_PROVIDERS` value. Export it in the shell you run
Terraform from, and that process will use your debugged provider rather than
starting its own.

## Self-hosted differences worth knowing

Not every route exists on a self-hosted install. The backups API in particular
is not routed at all, so requests to it return a 404 HTML page rather than an
API error — which surfaces as a confusing parse failure rather than a clean
"not supported". Dedicated databases, DocumentsDB and VectorsDB are Cloud
features.

TablesDB needs Appwrite 1.9.0 or later; earlier self-hosted versions answer its
routes with `general_route_not_found`.
