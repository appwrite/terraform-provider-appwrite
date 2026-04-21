#!/usr/bin/env bash
#
# Bootstrap a self-hosted Appwrite instance for acceptance testing.
#
# Usage:
#   ./testing/bootstrap.sh          # Start Appwrite + create project + run tests
#   ./testing/bootstrap.sh --up     # Just start Appwrite and print env vars
#   ./testing/bootstrap.sh --down   # Tear down the instance
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.yaml"
APPWRITE_URL="http://localhost:20080/v1"
COOKIE_JAR=$(mktemp)

cleanup() { rm -f "$COOKIE_JAR"; }
trap cleanup EXIT

# ── Helpers ──────────────────────────────────────────────────────────────────

log()  { echo "==> $*"; }
fail() { echo "ERROR: $*" >&2; exit 1; }

wait_for_appwrite() {
  log "Waiting for Appwrite to be ready..."
  local retries=60
  while [ $retries -gt 0 ]; do
    if curl -sf "$APPWRITE_URL/health" > /dev/null 2>&1; then
      log "Appwrite is ready!"
      return 0
    fi
    retries=$((retries - 1))
    sleep 2
  done
  fail "Appwrite did not become ready within 120 seconds"
}

api() {
  local method="$1" path="$2"
  shift 2
  curl -sf -X "$method" "$APPWRITE_URL$path" \
    -H "Content-Type: application/json" \
    -H "X-Appwrite-Project: console" \
    -b "$COOKIE_JAR" \
    -c "$COOKIE_JAR" \
    "$@"
}

# ── Commands ─────────────────────────────────────────────────────────────────

cmd_down() {
  log "Tearing down Appwrite..."
  docker compose -f "$COMPOSE_FILE" down -v --remove-orphans
  log "Done."
}

cmd_up() {
  log "Starting Appwrite..."
  docker compose -f "$COMPOSE_FILE" up -d --remove-orphans

  wait_for_appwrite

  # 1. Create admin account (first user = admin)
  log "Creating admin account..."
  api POST /account \
    -d '{"userId":"unique()","email":"admin@test.local","password":"password1234","name":"Admin"}' \
    > /dev/null 2>&1 || true

  # 2. Create session
  log "Logging in..."
  api POST /account/sessions/email \
    -d '{"email":"admin@test.local","password":"password1234"}' \
    > /dev/null || fail "Could not create session"

  # 3. Create a team (required for project)
  log "Creating team..."
  TEAM_ID=$(api POST /teams \
    -d '{"teamId":"unique()","name":"Testing"}' \
    2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin)['\$id'])" 2>/dev/null || echo "")

  if [ -z "$TEAM_ID" ]; then
    # Team may already exist from a previous run, fetch it
    TEAM_ID=$(api GET /teams \
      2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin)['teams'][0]['\$id'])" 2>/dev/null || echo "")
  fi

  [ -z "$TEAM_ID" ] && fail "Could not create or find a team"
  log "Team ID: $TEAM_ID"

  # 4. Create project
  log "Creating project..."
  PROJECT_ID=$(api POST /projects \
    -d "{\"projectId\":\"tf-test\",\"name\":\"Terraform Tests\",\"teamId\":\"$TEAM_ID\"}" \
    2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin)['\$id'])" 2>/dev/null || echo "")

  if [ -z "$PROJECT_ID" ]; then
    PROJECT_ID="tf-test"
    log "Project may already exist, using ID: $PROJECT_ID"
  else
    log "Project ID: $PROJECT_ID"
  fi

  # 5. Create API key with all scopes
  log "Creating API key..."
  ALL_SCOPES='["sessions.write","users.read","users.write","teams.read","teams.write","databases.read","databases.write","collections.read","collections.write","attributes.read","attributes.write","indexes.read","indexes.write","documents.read","documents.write","files.read","files.write","buckets.read","buckets.write","functions.read","functions.write","execution.read","execution.write","locale.read","health.read","avatars.read","webhooks.read","webhooks.write","rules.read","rules.write","messaging.read","messaging.write","providers.read","providers.write","topics.read","topics.write","subscribers.read","subscribers.write","targets.read","targets.write","backups.read","backups.write","sites.read","sites.write","vcs.read","vcs.write","migrations.read","migrations.write"]'

  API_KEY=$(api POST "/projects/$PROJECT_ID/keys" \
    -d "{\"name\":\"tf-acceptance-tests\",\"scopes\":$ALL_SCOPES}" \
    2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin)['secret'])" 2>/dev/null || echo "")

  if [ -z "$API_KEY" ]; then
    # Key may already exist, list keys
    API_KEY=$(api GET "/projects/$PROJECT_ID/keys" \
      2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin)['keys'][0]['secret'])" 2>/dev/null || echo "")
  fi

  [ -z "$API_KEY" ] && fail "Could not create or find an API key"

  # 6. Export env vars
  cat <<EOF

────────────────────────────────────────────────────────
  Appwrite is ready for acceptance testing!

  Export these variables:

    export APPWRITE_ENDPOINT="$APPWRITE_URL"
    export APPWRITE_PROJECT_ID="$PROJECT_ID"
    export APPWRITE_API_KEY="$API_KEY"

  Then run tests:

    make acceptance-test
────────────────────────────────────────────────────────
EOF

  # Write to .env file for convenience
  cat > "$SCRIPT_DIR/.env.test" <<EOF
APPWRITE_ENDPOINT=$APPWRITE_URL
APPWRITE_PROJECT_ID=$PROJECT_ID
APPWRITE_API_KEY=$API_KEY
EOF
  log "Env vars also saved to testing/.env.test"
}

cmd_test() {
  cmd_up

  # Source the env file and run tests
  log "Running acceptance tests..."
  set -a
  # shellcheck disable=SC1091
  source "$SCRIPT_DIR/.env.test"
  set +a

  cd "$SCRIPT_DIR/.."
  make acceptance-test
}

# ── Main ─────────────────────────────────────────────────────────────────────

case "${1:-}" in
  --down)  cmd_down ;;
  --up)    cmd_up ;;
  *)       cmd_test ;;
esac
