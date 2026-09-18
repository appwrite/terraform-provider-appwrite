#!/usr/bin/env bash
# Acceptance tests must go through acceptance.ResourceTest, which injects the
# destroy check, the no-replace plan guard, and refresh-after-apply.
#
# Without this check the guards are advisory: writing the obvious thing --
# resource.Test -- silently opts a new test out of all three, and nothing in
# review reliably catches it.
set -euo pipefail

# The wrapper itself is the one place allowed to call through.
allowed="internal/acceptance/testcase.go"

offenders=$(grep -rn --include='*.go' 'resource\.Test(\|resource\.ParallelTest(' internal \
  | grep -v "^${allowed}:" \
  || true)

if [ -n "$offenders" ]; then
  echo "Acceptance tests must call acceptance.ResourceTest, not resource.Test directly."
  echo
  echo "ResourceTest injects the destroy check, the no-replace plan guard and"
  echo "refresh-after-apply. A test that calls resource.Test gets none of them."
  echo "If the test deliberately exercises replacement, use"
  echo "acceptance.ResourceTestAllowingReplace instead."
  echo
  echo "$offenders"
  exit 1
fi

echo "OK: no direct resource.Test calls."
