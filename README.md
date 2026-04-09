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

| Resource                      | Description        |
|-------------------------------|--------------------|
| `appwrite_database`           | Database           |
| `appwrite_table`              | Database table     |
| `appwrite_column`             | Table column       |
| `appwrite_index`              | Table index        |
| `appwrite_bucket`             | Storage bucket     |
| `appwrite_messaging_provider` | Messaging provider |
| `appwrite_messaging_topic`    | Messaging topic    |
| `appwrite_user`               | User               |
| `appwrite_team`               | Team               |

## Data Sources

| Data Source         | Description                     |
|---------------------|---------------------------------|
| `appwrite_database` | Look up a database by identifier |

## Example

```hcl
resource "appwrite_database" "main" {
  id   = "main"
  name = "main"
}

resource "appwrite_table" "users" {
  database_id = appwrite_database.main.id
  id          = "users"
  name        = "users"
}

resource "appwrite_column" "name" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_table.users.id
  key         = "name"
  type        = "varchar"
  size        = 255
  required    = true
}

resource "appwrite_bucket" "images" {
  id                     = "images"
  name                   = "images"
  maximum_file_size       = 10485760
  allowed_file_extensions = ["jpg", "png", "webp", "gif"]
  compression            = "gzip"
  transformations        = true
}

resource "appwrite_messaging_provider" "sendgrid" {
  id         = "sendgrid"
  name       = "sendgrid"
  type       = "sendgrid"
  api_key    = "SG.test"
  from_email = "noreply@example.com"
  from_name  = "application"
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
