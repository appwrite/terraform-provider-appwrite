# Breaking changes

Breaking changes are not allowed within a major version. `v2.1.0` has shipped,
so anything on this list has to wait for `v3.0.0`, whatever else it would
improve.

This is not a style preference. A user pinned to `~> 2.0` receives minor and
patch releases automatically, so a breaking change in one of them fails their
next plan or apply with no action on their part, in whatever CI pipeline runs
next.

`TestSchemaHasNoBreakingChanges` compares the provider schema against
`.release/provider-schema.json` on every pull request and enforces most of what
follows mechanically.

## What breaks

- **Removing a resource, data source, ephemeral resource, or attribute.** State
  written by an earlier version refers to it.
- **Making an optional attribute required.** Every configuration that omitted it
  stops planning.
- **Removing `Computed`.** State already holds a server-assigned value, and
  without `Computed` the provider now plans to change it.
- **Making a settable attribute read-only.**
- **Changing an attribute's type.**
- **Removing `Sensitive`.** A value that used to be masked starts appearing in
  plan output and CI logs.
- **Making an attribute write-only.** It disappears from state, so anything
  referencing it breaks.
- **Lowering a schema version.** Terraform refuses to upgrade state it has
  already written past that point.
- **Tightening validation.** A configuration that applied cleanly now fails at
  plan time. This one is worth dwelling on: tightening validation *feels*
  like a bug fix, and it is the most commonly shipped accidental break.
- **Changing a default.** The next apply changes infrastructure nobody asked it
  to touch.
- **Any unexpected diff between two minor versions.** If `terraform plan` is
  empty on `2.1.0` and not empty on `2.2.0` for an unchanged configuration, that
  is a breaking change however small the cause.

## What does not break

- Adding a resource, data source, or ephemeral resource.
- Adding an optional or computed attribute.
- Relaxing an optional attribute's validation.
- Making a required attribute optional.
- Adding `Sensitive` to an attribute.
- Deprecating anything. A deprecation warning is not an error, which is the
  point of having one.

## Wanting to break something anyway

Deprecate it now, remove it in the next major version.

1. Mark the attribute or resource `DeprecationMessage`, naming the replacement.
   A message that says only "deprecated" leaves the reader to guess.
2. Add a `SchemaVersion` and an `UpgradeState` implementation if state has to be
   rewritten when the removal lands. Write it when you deprecate, not a year
   later when the shape of the old state is folklore.
3. Record it in the upgrade guide under `docs/guides/`.
4. Refresh the schema baseline in the same commit, so the diff shows a reviewer
   exactly what changed:

   ```sh
   APPWRITE_UPDATE_SCHEMA_BASELINE=1 go test ./internal/provider/ -run TestSchemaHasNoBreakingChanges
   ```

## Updating the baseline

Additions do not break anything but do make the baseline stale, so the test asks
for a refresh. Read the diff before committing it — that diff is the last place
an unintended break is cheap to catch.
