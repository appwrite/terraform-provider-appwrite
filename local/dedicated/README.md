# Dedicated database sandbox

Creates one dedicated database of **each** engine — PostgreSQL, MySQL and
MongoDB — plus the things that hang off them: backup policies, poolers, a
PostgreSQL extension, and optionally branches and a custom backup destination.

This directory is a separate Terraform root module with its own state, so it
does not touch the resources managed by the config in `local/`.

> [!WARNING]
> **This provisions real, billable infrastructure.** Three dedicated databases
> run until you destroy them. Use a scratch project, and run `terraform destroy`
> when you are done. A failed apply can leave a database behind that keeps
> billing, so check the Appwrite Console afterwards either way.

## Setup

Build the provider and point Terraform at your local build:

```bash
cd /Users/levi/source/terraform-provider-appwrite
make build
```

Add this to `~/.terraformrc` (or set `TF_CLI_CONFIG_FILE` to a file containing
it):

```hcl
provider_installation {
  dev_overrides {
    "appwrite/appwrite" = "/Users/levi/source/terraform-provider-appwrite"
  }
  direct {}
}
```

With `dev_overrides` in place, **do not run `terraform init`** — it will try to
fetch the provider from the registry and fail. Terraform prints a warning about
the override on every command; that warning is expected.

Point it at your Appwrite instance:

```bash
export APPWRITE_ENDPOINT="https://<host>/v1"
export APPWRITE_PROJECT_ID="<project-id>"
export APPWRITE_API_KEY="<standard project key>"
```

## Run it

**Plan first.** The specification data sources are read during planning, so a
plan costs nothing and still tells you two things worth knowing before you
spend: whether the endpoint actually exposes the dedicated database routes, and
which compute slug each database would be created on.

```bash
cd local/dedicated
terraform plan
```

If the routes are missing you will see `general_route_not_found` here rather
than after several minutes of provisioning.

```bash
terraform apply
```

Expect this to take **several minutes**. The provider waits for each database to
leave `provisioning` before returning, so dependent resources are never handed a
half-built database.

```bash
terraform destroy
```

## What it creates

| Resource | PostgreSQL | MySQL | MongoDB |
|---|---|---|---|
| `*_database` | yes | yes | yes |
| `*_backup_policy` | yes | yes | yes |
| `*_pooler` | yes | yes | none exists |
| `*_extension` | `pg_trgm` | n/a | n/a |
| `*_branch` | opt-in | opt-in | opt-in |
| `*_backup_storage` | opt-in | n/a | n/a |

It also reads back through `*_databases`, `*_database_status` and
`*_extensions` data sources, so one apply exercises most of the surface.

## Choosing a specification

Which compute slugs exist depends on the organization's billing plan, so the
config does not hardcode one. By default it reads the available slugs and picks
the **cheapest enabled** option per engine.

To see them all without creating anything:

```bash
terraform plan    # the plan shows the resolved specification per database
```

To pin one instead, copy `terraform.tfvars.example` to `terraform.tfvars` and
set `postgresql_specification` and friends.

## Keeping it cheap

`idle_timeout_minutes` defaults to `15`, so each database scales to zero after
fifteen minutes of inactivity. Set it to `0` for always-on, which costs more.

Branches (`create_branches = true`) and a custom backup destination
(`backup_storage = {...}`) are off by default because the first costs extra and
the second needs real bucket credentials.

## Notes

- `max_connections` is deliberately absent from the PostgreSQL pooler: it is
  read-only there, and setting it fails at plan time.
- `appwrite_postgresql_backup_storage` has no read route on the API, so
  Terraform cannot detect drift on it, cannot import it, and destroying it only
  removes it from state — the server keeps writing backups where it was last
  told to.
