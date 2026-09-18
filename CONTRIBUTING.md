# Contributing to Terraform Provider for Appwrite

Thank you for your interest in contributing! This document provides guidelines for contributing to the Appwrite Terraform provider.

## Development Setup

### Prerequisites

- [Go](https://golang.org/doc/install) (see `go.mod` for required version)
- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- An Appwrite instance (Cloud or Community Edition) for acceptance testing

### Building

```bash
make build
```

### Installing Locally

```bash
make install
```

This installs the provider to `~/.terraform.d/plugins/` for local testing.

### Running Tests

Unit tests:

```bash
make test
```

Every pull request runs the acceptance suite in CI against an Appwrite brought
up on the runner, so a change that only breaks against a real server fails on
the pull request. Locally the suite needs an instance of your own:

```bash
export APPWRITE_ENDPOINT="https://cloud.appwrite.io/v1"
export APPWRITE_PROJECT_ID="your-project-id"
# Standard project key for project-scoped tests:
export APPWRITE_API_KEY="your-project-api-key"
# Required only for organization-scoped tests such as appwrite_project and
# appwrite_project_key. Use an organization key with projects/keys scopes:
export APPWRITE_ORGANIZATION_ID="your-organization-id"
export APPWRITE_ORGANIZATION_API_KEY="your-organization-api-key"
# Required for proxy-rule tests on Cloud. The domain must be owned by the
# organization; tests create unique subdomains beneath it:
export APPWRITE_TEST_DOMAIN="example.com"
make acceptance-test
```

#### Dedicated database tests

Dedicated database tests provision real, billable infrastructure and take
several minutes per step, so they are skipped unless explicitly enabled:

```bash
# Opt in. Without this, every dedicated database test skips:
export APPWRITE_DEDICATED_DATABASE_TESTS=1
# Optional. The compute slug to provision with; which slugs exist depends on the
# organization's billing plan. Defaults to s-1vcpu-1gb:
export APPWRITE_DEDICATED_SPECIFICATION="s-1vcpu-1gb"

make acceptance-test TESTARGS='-run TestAccPostgresql'
```

Run them against a scratch project. A failed run can leave a database behind,
which keeps billing until it is deleted; check the Console afterwards.

#### Project API key tests

Appwrite removed the endpoint for creating project API keys, so
`appwrite_project_key` cannot create one and its tests work against a key you
create in the Console first:

```bash
export APPWRITE_PROJECT_KEY_ID="existing-key-id"
```

Without it, the import/update test skips; the test asserting that creation fails
runs either way.

### Linting

```bash
make lint
```

`make lint` runs `go vet`, the formatting check, and golangci-lint at the version
pinned in the Makefile. Pinned deliberately: a floating linter changes what
passes review without anyone committing anything.

### Running everything CI runs

```bash
make ci
```

This is the same set CI runs, in one target, so a green local run means a green
pull request. It covers linting, the repository checks, `go mod tidy`, the unit
suite, both sweeper-linkage assertions, and the documentation diff.

`make depscheck` compares the working tree against the index, so it reports a
difference whenever `go.mod` has uncommitted changes -- including correct ones.
That is intentional for CI, where the committed `go.mod` is what matters.

### Generating Documentation

```bash
make docs
```

Documentation is auto-generated from schema definitions and examples using [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs). Always run `make docs` after changing resource schemas or examples.

## Adding a New Resource

1. Create a new directory under `internal/services/<service>/`
2. Implement `resource.go` with the Plugin Framework interfaces
3. Add the resource to `provider.go` in the `Resources()` method
4. Create an example in `examples/resources/appwrite_<resource_name>/resource.tf`
5. Create an import example in `examples/resources/appwrite_<resource_name>/import.sh`
6. Create a doc template in `templates/resources/<resource_name>.md.tmpl`
7. Write acceptance tests in `resource_test.go`, calling
   `acceptance.ResourceTest` rather than `resource.Test` -- see below
8. Register a destroy check for the new type in
   `internal/acceptance/destroy.go`, or record why it cannot have one
9. Add a sweeper in `internal/sweep/sweepers.go` if the resource can be left
   behind by an interrupted run
10. Append `common.ForcesReplacementNote` to the description of every argument
    that forces replacement
11. Run `make docs` to generate documentation
12. Refresh the schema baseline:
    `APPWRITE_UPDATE_SCHEMA_BASELINE=1 go test ./internal/provider/`

### Tests must go through the shared harness

Acceptance tests call `acceptance.ResourceTest`, not `resource.Test`. The wrapper
injects three things no test should be without:

- a **destroy check** against the server. The framework only verifies that
  Terraform removed the resource from state, so a `Delete` that returns success
  without deleting anything passes an unguarded test.
- a **no-replace plan check** on every configuration step, so a resource cannot
  quietly start recreating itself. If the test's subject *is* replacement, use
  `acceptance.ResourceTestAllowingReplace`.
- **refresh-after-apply**, which catches `Create` writing a value to state that
  `Read` does not produce -- otherwise seen only as a perpetual diff for users.

`scripts/checks/no-direct-resource-test.sh` enforces this, because a guard that
can be bypassed by writing the obvious thing instead is not a guard.

### Sweepers

`internal/sweep` deletes resources an interrupted run left behind. It only
touches resources whose name or ID contains `tf-acc-test`, and it refuses to run
unless `APPWRITE_SWEEP_PROJECT_ID` names the project explicitly -- it will not
fall back to `APPWRITE_PROJECT_ID`, because that holds whatever the last run
exported.

```bash
APPWRITE_SWEEP_PROJECT_ID=<disposable project> make sweep
```

Sweeper code must stay reachable only from test files. `make sweeper-unlinked`
asserts it is absent from the provider binary users install.

## Adding a New Data Source

Follow the same pattern as resources, but use `datasource.DataSource` interface and register in the `DataSources()` method.

## Pull Request Process

1. Fork the repository and create a feature branch
2. Write or update tests for your changes
3. Run `make ci` to verify them the way CI will
4. Run `make docs` and commit any generated documentation changes
5. Add a `CHANGELOG.md` entry, or apply the `skip-changelog` label if the change
   genuinely alters nothing a user would notice
6. Open a pull request with a clear description of the changes

### Breaking changes

Breaking changes are not allowed within a major version, and `v2.1.0` has
shipped. `contributing/breaking-changes.md` has the full list of what counts and
the deprecate-then-remove sequence to follow instead. The schema is compared
against a committed baseline on every pull request, so most of it is enforced
rather than remembered.

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use the Terraform Plugin Framework (not SDK v2) for new resources
- Keep resource implementations consistent with existing patterns
- Mark sensitive fields with `Sensitive: true`
- Handle 404 errors in Read by calling `resp.State.RemoveResource(ctx)`
- Match errors on Appwrite's structured `type` field via `common.ErrorType`, not
  on the prose of the message. Messages get reworded; types are contractual
- Every attribute needs a description, ending in a full stop
- Wrap errors with `%w`, and do not begin a wrapped message with "failed" or
  "error" -- a caller will prefix it
