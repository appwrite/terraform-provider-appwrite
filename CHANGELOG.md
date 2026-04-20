# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Data sources for storage buckets, users, teams, functions, sites, topics, and webhooks
- Schema validators for deployment `source_type`, bucket `compression`, and function `name`/`runtime`
- User-Agent header (`terraform-provider-appwrite/<version>`) on all API calls
- Acceptance tests for function and site deployment resources and data sources
- `CHANGELOG.md`, `CONTRIBUTING.md`, `SECURITY.md`, and GitHub issue templates
- `CODEOWNERS` file and pull request template
- `golangci-lint` in CI pipeline
- Import support for function and site deployment resources

## [1.2.1] - 2026-04-17

### Fixed

- Index creation resource handling
- Duplicate example files removed from the repository

## [1.2.0] - 2026-04-17

### Changed

- Upgraded `go-sdk` to `v3.0.0` with webhook field renames

## [1.1.0] - 2026-04-17

### Added

- Deployment resources for sites and functions
- Documentation and examples for deployment resources

## [1.0.2] - 2026-04-17

### Changed

- Polished repository README

### Fixed

- Added `WaitForColumnAvailable` preventing column creation errors
- Error handling for self-hosted API limitations

## [1.0.1] - 2026-04-13

### Changed

- Refactored documentation

## [1.0.0] - 2026-04-12

### Added

- Resources for functions and sites with their respective variables
- Cleaned up documentation

## [0.0.8] - 2026-04-12

### Added

- Initial release with support for TablesDB, Storage, Auth, Messaging, Webhooks, and Backup Policies
- Acceptance tests for all resources
- Auto-generated documentation via `terraform-plugin-docs`
