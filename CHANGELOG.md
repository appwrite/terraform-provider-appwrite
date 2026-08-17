# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Dedicated database support, backed by the new `Postgresql`, `Mysql` and `Mongo` SDK services. Each engine gets its own resource and data source because Appwrite routes them separately and only some engines have a pooler or extensions:
  - `appwrite_postgresql_database`, `appwrite_mysql_database` and `appwrite_mongo_database` resources, covering provisioning, in-place resize, major version upgrade, replicas, point-in-time recovery, storage autoscaling, network allowlists, idle timeouts, maintenance windows and the SQL API. Create and update wait for the database to leave its transitional state, so dependent resources are never handed a half-built database
  - `appwrite_postgresql_backup_policy`, `appwrite_mysql_backup_policy` and `appwrite_mongo_backup_policy` resources for scheduled backups of a dedicated database
  - `appwrite_postgresql_pooler` and `appwrite_mysql_pooler` resources for connection pooling
  - `appwrite_postgresql_extension` resource for installing PostgreSQL extensions
  - `appwrite_postgresql_database`, `appwrite_mysql_database` and `appwrite_mongo_database` data sources
  - `appwrite_postgresql_specifications`, `appwrite_mysql_specifications` and `appwrite_mongo_specifications` data sources, so a compute slug can be selected from what the billing plan allows instead of hardcoded
  - `appwrite_postgresql_extensions` data source listing installed and available extensions
- `appwrite_proxy_rule` resource for site and function custom domains
- `appwrite_project` resource for organization-scoped project provisioning
- `appwrite_project_key` resource for managing project API keys. Keys are created in the Appwrite Console and imported; see the note under Changed
- Provider-level `organization_id` configuration with `APPWRITE_ORGANIZATION_ID` support
- Separate `organization_api_key` configuration with `APPWRITE_ORGANIZATION_API_KEY` support
- Resource-specific credential type validation and authentication guidance

### Changed

- Upgraded `go-sdk` to `v7.2.0-rc.1`
- **Breaking:** `appwrite_project_key` can no longer create keys. Appwrite removed the create-project-key endpoint so that a leaked API key cannot mint further keys and outlive its own revocation, and there is no server-side replacement. A plan that would create a key now fails with guidance to create it in the Console and `terraform import` it; read, update, delete and import are unaffected. The `secret` attribute is only populated for keys created before this change, since the API only returns a secret at creation time

### Fixed

- `appwrite_tablesdb_column`: reading a column no longer fails against SDK v6.5.0 and later, where a JSON response body is `[]byte` rather than `string`
- `appwrite_tablesdb_column`: `size` is no longer ignored for `type = "string"` — create/update call the string column endpoints again instead of the text endpoints, so columns materialize as `string(size)` (indexable) rather than unbounded TEXT ([#31](https://github.com/appwrite/terraform-provider-appwrite/issues/31))

## [1.6.0] - 2026-06-04

### Changed

- Upgraded `go-sdk` to `v5.0.0`

## [1.5.0] - 2026-05-27

### Added

- Plan modifier for column default values

### Changed

- Documentation now mentions TablesDB version compatibility

## [1.4.0] - 2026-05-19

### Changed

- Upgraded `go-sdk` to `v4.0.0`

## [1.3.2] - 2026-05-11

### Added

- `bigint` column support

### Changed

- Upgraded `sdk-for-go` to `v3.1.0`

## [1.3.1] - 2026-05-08

### Fixed

- Populate column `type` from the API during resource import

## [1.3.0] - 2026-04-20

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

## [1.0.0] - 2026-04-11

### Added

- Resources for functions and sites with their respective variables
- Cleaned up documentation

## [0.0.8] - 2026-04-10

### Added

- In-place updates for resources

## [0.0.7] - 2026-04-10

### Added

- `storage_file`, `messaging_subscriber`, and `webhook` resources

## [0.0.6] - 2026-04-09

### Added

- `appwrite_auth_user` resource (name, email, phone, password, status, labels, verification)
- `appwrite_auth_team` resource (name, roles)
- `appwrite_backup_policy` resource (services, retention, schedule, resource targeting)
- `appwrite_tablesdb_row` resource (JSON data, permissions)
- Optional `id` on all resources — Appwrite auto-generates an ID when omitted
- Per-resource documentation templates with "See Also" sections
- Cross-resource examples (team-scoped messaging topics, backup policies tied to specific databases)

### Changed

- **Breaking:** Resources prefixed with their service category to support future database types:
  - `appwrite_database` → `appwrite_tablesdb`
  - `appwrite_table` → `appwrite_tablesdb_table`
  - `appwrite_column` → `appwrite_tablesdb_column`
  - `appwrite_index` → `appwrite_tablesdb_index`
  - `appwrite_bucket` → `appwrite_storage_bucket`
  - `appwrite_user` → `appwrite_auth_user`
  - `appwrite_team` → `appwrite_auth_team`
  - `data.appwrite_database` → `data.appwrite_tablesdb`

### Fixed

- User labels vanishing on create — provider now performs a final read after post-create updates

## [0.0.5] - 2026-04-09

### Added

- Appwrite auth `user` and `team` resources

## [0.0.4] - 2026-04-08

### Added

- Appwrite messaging `provider` and `topic` resources

## [0.0.3] - 2026-04-08

### Added

- Appwrite storage `bucket` resource

## [0.0.2] - 2026-04-08

### Fixed

- Automatic release process to the public Terraform registry

## [0.0.1] - 2026-04-08

### Added

- Initial release of the Appwrite Terraform Provider (supports Appwrite Cloud and Community Edition)
- Resources: `appwrite_database`, `appwrite_table`, `appwrite_column`, `appwrite_index`
- Column types: varchar, text, longtext, mediumtext, integer, float, boolean, enum, email, datetime, url, ip, point, line, polygon, relationship, string
- Index types: key, unique, and fulltext; automatically waits for columns to become available
- Data source: `appwrite_database`
- Provider configuration: `endpoint`, `project_id`, `api_key`, `self_signed` (with matching `APPWRITE_*` env vars)
- Import support for all resources
