# Terraform Provider for Appwrite

A Terraform provider for managing [Appwrite](https://appwrite.io) resources. Works with both Appwrite Cloud and Community Edition.

## Setup

Appwrite Cloud:

```hcl
terraform {
  required_providers {
    appwrite = {
      source = "appwrite/appwrite"
    }
  }
}

provider "appwrite" {
  endpoint   = "https://cloud.appwrite.io/v1"
  project_id = "project-id"
  api_key    = "api-key"
}
```

Appwrite Community Edition:

```hcl
provider "appwrite" {
  endpoint    = "https://appwrite-instance.com/v1"
  project_id  = "project-id"
  api_key     = "api-key"
  self_signed = true
}
```

All provider attributes can also be set via environment variables:

```bash
export APPWRITE_ENDPOINT="https://cloud.appwrite.io/v1"
export APPWRITE_PROJECT_ID="project-id"
export APPWRITE_API_KEY="api-key"
```

## Provider Options

If an environment variable is provided, then the option does not need to be set in the terraform provider configuration.

| Property    | Environment Variable   | Required | Description                                            |
|-------------|------------------------|----------|--------------------------------------------------------|
| endpoint    | `APPWRITE_ENDPOINT`    | yes      | The Appwrite API endpoint                              |
| project_id  | `APPWRITE_PROJECT_ID`  | yes      | The Appwrite project ID                                |
| api_key     | `APPWRITE_API_KEY`     | yes      | The Appwrite API key                                   |
| self_signed | —                      | no       | Accept self-signed certificates (for community edition)|

## Resources

| Resource            | Description             |
|---------------------|-------------------------|
| `appwrite_database` | Database                |
| `appwrite_table`    | Table within a database |
| `appwrite_column`   | Column within a table   |
| `appwrite_index`    | Table index             |

## Data Sources

| Data Source         | Description              |
|---------------------|--------------------------|
| `appwrite_database` | Look up a database by ID |

## Example

```hcl
resource "appwrite_database" "main" {
  id   = "main"
  name = "Main"
}

resource "appwrite_table" "users" {
  database_id = appwrite_database.main.id
  id          = "users"
  name        = "Users"
}

resource "appwrite_column" "name" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_table.users.id
  key         = "name"
  type        = "varchar"
  size        = 255
  required    = true
}

resource "appwrite_column" "email" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_table.users.id
  key         = "email"
  type        = "email"
  required    = true
}

resource "appwrite_column" "age" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_table.users.id
  key         = "age"
  type        = "integer"
  min         = 0
  max         = 150
}

resource "appwrite_column" "role" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_table.users.id
  key         = "role"
  type        = "enum"
  elements    = ["admin", "editor", "viewer"]
  default     = "viewer"
}

resource "appwrite_column" "tags" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_table.users.id
  key         = "tags"
  type        = "varchar"
  size        = 64
  array       = true
}

resource "appwrite_column" "location" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_table.users.id
  key         = "location"
  type        = "point"
}

resource "appwrite_index" "email_unique" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_table.users.id
  key         = "email_unique"
  type        = "unique"
  columns     = [appwrite_column.email.key]
}
```

## Development

```bash
make build              # build the provider
make install            # install to local terraform plugins
make test               # run unit tests
make acceptance-test    # run acceptance tests (requires Appwrite credentials)
make lint               # run go vet + format check
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
