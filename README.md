# Terraform Provider for Appwrite

A Terraform provider for managing [Appwrite](https://appwrite.io) resources. Works with both Appwrite Cloud and Community Edition.

## Setup

```hcl
terraform {
  required_providers {
    appwrite = {
      source = "appwrite/appwrite"
    }
  }
}

provider "appwrite" {
  endpoint             = "https://cloud.appwrite.io/v1"
  project_id           = "project-id"
  organization_id      = "organization-id"
  api_key              = "project-api-key"
  organization_api_key = "organization-api-key"
}
```

Configure credentials via environment variables:

```bash
export APPWRITE_ENDPOINT="https://cloud.appwrite.io/v1"
export APPWRITE_PROJECT_ID="project-id"
export APPWRITE_ORGANIZATION_ID="organization-id"
export APPWRITE_API_KEY="project-api-key"
export APPWRITE_ORGANIZATION_API_KEY="organization-api-key"
```

## Provider Options

| Property               | Environment Variable               | Required | Description                                                          |
|------------------------|------------------------------------|----------|----------------------------------------------------------------------|
| endpoint               | `APPWRITE_ENDPOINT`                | yes      | The Appwrite API endpoint                                            |
| project_id             | `APPWRITE_PROJECT_ID`              | no       | The default Appwrite project ID                                      |
| organization_id        | `APPWRITE_ORGANIZATION_ID`         | no       | The default Appwrite organization ID                                 |
| api_key                | `APPWRITE_API_KEY`                 | yes      | Project API key for project-scoped resources                         |
| organization_api_key   | `APPWRITE_ORGANIZATION_API_KEY`    | no       | Organization API key; defaults to `api_key` for backwards compatibility |
| self_signed            | N/A                                | no       | Accept self-signed certificates (for Community Edition)              |

## Compatibility

TablesDB resources are intended for Appwrite Cloud or Appwrite Community Edition 1.9.0 and later. Earlier self-hosted versions may return `general_route_not_found` errors for TablesDB routes.

Dedicated database resources require a server that exposes the `/postgresql`, `/mysql` and `/mongo` routes, and the available compute specifications depend on the organization's billing plan. Read the slugs from the matching `*_specifications` data source rather than hardcoding one.

## Resources

### Projects

| Resource                     | Description     |
|------------------------------|-----------------|
| `appwrite_project`           | Project         |
| `appwrite_project_key`       | Project API key |

Appwrite removed the endpoint for creating project API keys, so `appwrite_project_key` cannot create one — a key able to mint further keys would let a compromise outlive the revocation of the leaked key. Create the key in the Console and `terraform import` it; read, update and delete work as before.

### Proxy

| Resource                  | Description                         |
|---------------------------|-------------------------------------|
| `appwrite_proxy_rule`     | Site or function custom domain rule |

### TablesDB

| Resource                    | Description    |
|-----------------------------|----------------|
| `appwrite_tablesdb`         | Database       |
| `appwrite_tablesdb_table`   | Table          |
| `appwrite_tablesdb_column`  | Column         |
| `appwrite_tablesdb_index`   | Index          |
| `appwrite_tablesdb_row`     | Row            |

### Dedicated Databases

Databases running on infrastructure reserved for a single project, as opposed to
TablesDB's shared infrastructure. Provisioning, resizing and upgrading take
several minutes; the provider waits for the database to settle before continuing.

| Resource                             | Description                           |
|--------------------------------------|---------------------------------------|
| `appwrite_postgresql_database`       | Dedicated PostgreSQL database         |
| `appwrite_mysql_database`            | Dedicated MySQL database              |
| `appwrite_mongo_database`            | Dedicated MongoDB database            |
| `appwrite_postgresql_backup_policy`  | Scheduled backup policy (PostgreSQL)  |
| `appwrite_mysql_backup_policy`       | Scheduled backup policy (MySQL)       |
| `appwrite_mongo_backup_policy`       | Scheduled backup policy (MongoDB)     |
| `appwrite_postgresql_backup_storage` | Custom backup destination (PostgreSQL)|
| `appwrite_mysql_backup_storage`      | Custom backup destination (MySQL)     |
| `appwrite_mongo_backup_storage`      | Custom backup destination (MongoDB)   |
| `appwrite_postgresql_branch`         | Database branch (PostgreSQL)          |
| `appwrite_mysql_branch`              | Database branch (MySQL)               |
| `appwrite_mongo_branch`              | Database branch (MongoDB)             |
| `appwrite_postgresql_pooler`         | Connection pooler (PostgreSQL)        |
| `appwrite_mysql_pooler`              | Connection pooler (MySQL)             |
| `appwrite_postgresql_extension`      | Installed PostgreSQL extension        |

Day-2 operations that do not model as declarative state — failover, migration,
restoration, on-demand backups, credential rotation and running SQL — are not
exposed as resources; run them through the Console or API. `*_backup_storage`
has no read route on the API, so Terraform cannot detect drift on it or import
an existing configuration.

### Storage

| Resource                  | Description    |
|---------------------------|----------------|
| `appwrite_storage_bucket` | Bucket         |
| `appwrite_storage_file`   | File           |

### Auth

| Resource             | Description |
|----------------------|-------------|
| `appwrite_auth_user` | User        |
| `appwrite_auth_team` | Team        |

### Functions

| Resource                          | Description          |
|-----------------------------------|----------------------|
| `appwrite_function`               | Function             |
| `appwrite_function_variable`      | Environment variable |
| `appwrite_function_deployment`    | Deployment           |

### Sites

| Resource                      | Description          |
|-------------------------------|----------------------|
| `appwrite_site`               | Site                 |
| `appwrite_site_variable`      | Environment variable |
| `appwrite_site_deployment`    | Deployment           |

### Messaging

| Resource                        | Description |
|---------------------------------|-------------|
| `appwrite_messaging_provider`   | Provider    |
| `appwrite_messaging_topic`      | Topic       |
| `appwrite_messaging_subscriber` | Subscriber  |

### Other

| Resource                 | Description   |
|--------------------------|---------------|
| `appwrite_webhook`       | Webhook       |
| `appwrite_backup_policy` | Backup policy |

## Data Sources

| Data Source                          | Description                                        |
|--------------------------------------|----------------------------------------------------|
| `appwrite_tablesdb`                  | Look up a database by ID                           |
| `appwrite_postgresql_database`       | Look up a dedicated PostgreSQL database by ID      |
| `appwrite_mysql_database`            | Look up a dedicated MySQL database by ID           |
| `appwrite_mongo_database`            | Look up a dedicated MongoDB database by ID         |
| `appwrite_postgresql_databases`      | List dedicated PostgreSQL databases                |
| `appwrite_mysql_databases`           | List dedicated MySQL databases                     |
| `appwrite_mongo_databases`           | List dedicated MongoDB databases                   |
| `appwrite_postgresql_specifications` | List available PostgreSQL compute specifications   |
| `appwrite_mysql_specifications`      | List available MySQL compute specifications        |
| `appwrite_mongo_specifications`      | List available MongoDB compute specifications      |
| `appwrite_postgresql_database_status`| Live health, replication and storage (PostgreSQL)  |
| `appwrite_mysql_database_status`     | Live health, replication and storage (MySQL)       |
| `appwrite_mongo_database_status`     | Live health, replication and storage (MongoDB)     |
| `appwrite_postgresql_backups`        | List PostgreSQL backups                            |
| `appwrite_mysql_backups`             | List MySQL backups                                 |
| `appwrite_mongo_backups`             | List MongoDB backups                               |
| `appwrite_postgresql_extensions`     | List installed and available PostgreSQL extensions |

## Example

```hcl
resource "appwrite_tablesdb" "main" {
  id   = "main"
  name = "main"
}

resource "appwrite_tablesdb_table" "users" {
  database_id = appwrite_tablesdb.main.id
  id          = "users"
  name        = "users"
}

resource "appwrite_tablesdb_column" "name" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "name"
  type        = "varchar"
  size        = 255
  required    = true
}

resource "appwrite_function" "on_signup" {
  name       = "on-signup"
  runtime    = "node-22"
  entrypoint = "index.js"
  commands   = "npm install"
  events     = ["users.*.create"]
  timeout    = 30
}

resource "appwrite_function_variable" "api_url" {
  function_id = appwrite_function.on_signup.id
  key         = "API_URL"
  value       = "https://api.example.com"
}

resource "appwrite_site" "dashboard" {
  name            = "dashboard"
  framework       = "nextjs"
  build_runtime   = "node-22"
  install_command = "npm install"
  build_command   = "npm run build"
}

resource "appwrite_site_variable" "api_url" {
  site_id = appwrite_site.dashboard.id
  key     = "NEXT_PUBLIC_API_URL"
  value   = "https://api.example.com"
}

resource "appwrite_function_deployment" "on_signup" {
  function_id = appwrite_function.on_signup.id
  source_type = "code"
  code_path   = "./on-signup.tar.gz"
  code_hash   = filesha256("./on-signup.tar.gz")
  activate    = true
}

resource "appwrite_site_deployment" "dashboard" {
  site_id        = appwrite_site.dashboard.id
  source_type    = "template"
  repository     = "templates-for-sites"
  owner          = "appwrite"
  root_directory = "nextjs/starter"
  type           = "branch"
  reference      = "main"
  activate       = true
}

resource "appwrite_storage_bucket" "images" {
  id                      = "images"
  name                    = "images"
  maximum_file_size       = 10485760
  allowed_file_extensions = ["jpg", "png", "webp", "gif"]
  compression             = "gzip"
  transformations         = true
}

resource "appwrite_webhook" "user_events" {
  name   = "user-events"
  url    = "https://api.example.com/webhooks/users"
  events = ["users.*.create", "users.*.update"]
}
```

## Development

```bash
make build              # build the provider
make install            # install to local terraform plugins
make test               # run unit tests
make acceptance-test    # run acceptance tests (requires Appwrite credentials)
make lint               # run go vet + format check
make docs               # generate documentation
make fmt                # format all go files
make clean              # remove built binary
```

For local development, add a dev override to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "appwrite/appwrite" = "/path/to/terraform-provider"
  }
  direct {}
}
```
