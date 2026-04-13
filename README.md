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

provider "appwrite" {}
```

Configure credentials via environment variables:

```bash
export APPWRITE_ENDPOINT="https://cloud.appwrite.io/v1"
export APPWRITE_PROJECT_ID="project-id"
export APPWRITE_API_KEY="api-key"
```

Or set them directly in the provider block:

```hcl
provider "appwrite" {
  endpoint   = "https://cloud.appwrite.io/v1"
  project_id = "project-id"
  api_key    = "api-key"
}
```

## Provider Options

| Property    | Environment Variable   | Required | Description                                             |
|-------------|------------------------|----------|---------------------------------------------------------|
| endpoint    | `APPWRITE_ENDPOINT`    | yes      | The Appwrite API endpoint                               |
| project_id  | `APPWRITE_PROJECT_ID`  | yes      | The Appwrite project ID                                 |
| api_key     | `APPWRITE_API_KEY`     | yes      | The Appwrite API key                                    |
| self_signed | N/A                    | no       | Accept self-signed certificates (for Community Edition) |

## Resources

### TablesDB

| Resource                    | Description    |
|-----------------------------|----------------|
| `appwrite_tablesdb`         | Database       |
| `appwrite_tablesdb_table`   | Table          |
| `appwrite_tablesdb_column`  | Column         |
| `appwrite_tablesdb_index`   | Index          |
| `appwrite_tablesdb_row`     | Row            |

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

| Resource                       | Description          |
|--------------------------------|----------------------|
| `appwrite_function`            | Function             |
| `appwrite_function_variable`   | Environment variable |

### Sites

| Resource                   | Description          |
|----------------------------|----------------------|
| `appwrite_site`            | Site                 |
| `appwrite_site_variable`   | Environment variable |

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

| Data Source         | Description              |
|---------------------|--------------------------|
| `appwrite_tablesdb` | Look up a database by ID |

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
