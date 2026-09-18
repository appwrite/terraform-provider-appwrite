#!/usr/bin/env bash
# The 75 example .tf files under examples/ are the code users copy first, and
# nothing currently checks them. An example that does not parse, or that is
# formatted differently from what `terraform fmt` produces, teaches the wrong
# thing before the provider has done anything at all.
#
# Formatting only. `terraform validate` would need a provider installed from the
# registry, which means it would validate the examples against the last release
# rather than against the code in the pull request -- worse than useless when an
# example is being added for a new resource.
set -euo pipefail

if ! command -v terraform >/dev/null 2>&1; then
  if [ -n "${REQUIRE_TERRAFORM:-}" ]; then
    echo "terraform is required for this check but was not found on PATH" >&2
    exit 1
  fi
  echo "SKIP: terraform not on PATH, cannot check example formatting."
  echo "      CI sets REQUIRE_TERRAFORM=1 so this cannot be skipped there."
  exit 0
fi

if ! terraform fmt -check -recursive -diff examples/; then
  echo
  echo "Example configurations are not formatted. Run:"
  echo "    terraform fmt -recursive examples/"
  exit 1
fi

echo "OK: example configurations are formatted."
