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

Acceptance tests (requires a running Appwrite instance):

```bash
export APPWRITE_ENDPOINT="https://cloud.appwrite.io/v1"
export APPWRITE_PROJECT_ID="your-project-id"
export APPWRITE_API_KEY="your-api-key"
# Required only for organization-scoped tests such as appwrite_project and
# appwrite_project_key. Use an organization key with projects/keys scopes:
export APPWRITE_ORGANIZATION_ID="your-organization-id"
make acceptance-test
```

### Linting

```bash
make lint
```

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
7. Write acceptance tests in `resource_test.go`
8. Run `make docs` to generate documentation

## Adding a New Data Source

Follow the same pattern as resources, but use `datasource.DataSource` interface and register in the `DataSources()` method.

## Pull Request Process

1. Fork the repository and create a feature branch
2. Write or update tests for your changes
3. Run `make lint` and `make test` to verify your changes
4. Run `make docs` and commit any generated documentation changes
5. Open a pull request with a clear description of the changes

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use the Terraform Plugin Framework (not SDK v2) for new resources
- Keep resource implementations consistent with existing patterns
- Mark sensitive fields with `Sensitive: true`
- Handle 404 errors in Read by calling `resp.State.RemoveResource(ctx)`
