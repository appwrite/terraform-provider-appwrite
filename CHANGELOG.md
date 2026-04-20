# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Data sources for storage buckets, users, teams, functions, sites, topics, and webhooks
- Schema validators for deployment `source_type`, bucket `compression`, and function `name`/`runtime`
- User-Agent header (`terraform-provider-appwrite/<version>`) on all API calls
- Acceptance tests for function and site deployment resources
- Examples and documentation for deployment resources
- `CHANGELOG.md`, `CONTRIBUTING.md`, `SECURITY.md`, and GitHub issue templates
- `golangci-lint` in CI pipeline

### Removed

- Empty `projectvar` package

## [0.1.0] - 2026-04-12

### Added

- Initial release with 20 resources and 1 data source
- Support for TablesDB (databases, tables, columns, indexes, rows)
- Support for Storage (buckets, files)
- Support for Auth (users, teams)
- Support for Functions (functions, variables, deployments)
- Support for Sites (sites, variables, deployments)
- Support for Messaging (topics, providers, subscribers)
- Support for Webhooks and Backup Policies
- Acceptance tests for all resources
- Auto-generated documentation via `terraform-plugin-docs`
