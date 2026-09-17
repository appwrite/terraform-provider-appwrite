#!/usr/bin/env bash
#
# Seeds the Appwrite started by appwrite-up.sh and exports the credentials the
# acceptance tests read. Writes to $GITHUB_ENV when running under Actions, and
# prints shell exports otherwise so the same script seeds a local instance.
#
# The sequence mirrors what the Console does on a fresh install: sign up the
# first account, open a session, take the organization, create a project under
# it, then mint a key for that project.
#
# Two things about admin mode are easy to get wrong:
#   - project-scoped admin routes (minting a key) need X-Appwrite-Mode: admin
#     alongside the session cookie, or the caller is treated as a guest;
#   - the console project rejects that same header outright ("Admin mode is not
#     allowed for console project"), so console-scoped calls must omit it.
set -euo pipefail

ENDPOINT="${APPWRITE_ENDPOINT_BASE:-http://localhost}"
CONTAINER="${APPWRITE_CONTAINER:-appwrite}"
COOKIES="$(mktemp)"
trap 'rm -f "$COOKIES"' EXIT

if ! command -v jq >/dev/null 2>&1; then
  echo "::error::jq is required by this script" >&2
  exit 1
fi

EMAIL="ci-$(date +%s)@terraform-provider-appwrite.test"
PASSWORD="$(head -c 24 /dev/urandom | base64 | tr -d '/+=')Aa1!"

# Usage: api METHOD PATH JSON_BODY [extra curl args...]
api() {
  local method="$1" path="$2" data="$3"
  shift 3
  curl -fsS -m 30 -X "$method" "${ENDPOINT}/v1${path}" \
    -b "$COOKIES" -c "$COOKIES" \
    -H 'Content-Type: application/json' \
    -H 'Accept: application/json' \
    -d "$data" \
    "$@"
}

require() {
  local value="$1" what="$2"
  if [ -z "$value" ] || [ "$value" = "null" ]; then
    echo "::error::Failed to determine ${what}" >&2
    exit 1
  fi
}

echo "Creating console account..."
api POST /account "$(jq -nc --arg e "$EMAIL" --arg p "$PASSWORD" \
  '{userId:"unique()",email:$e,password:$p,name:"Terraform CI"}')" \
  -H 'X-Appwrite-Project: console' > /dev/null

echo "Opening console session..."
api POST /account/sessions/email "$(jq -nc --arg e "$EMAIL" --arg p "$PASSWORD" \
  '{email:$e,password:$p}')" \
  -H 'X-Appwrite-Project: console' > /dev/null

# A self-hosted instance permits exactly one organization, created by whoever
# signs up first. On the fresh instance CI boots, that is this account.
#
# Re-running against an instance that already has an organization does not work:
# this run signs up a new account, which is not a member of that organization,
# cannot see it, and is refused when it tries to create one ("This self-hosted
# instance already has an organization"). Start from a clean instance instead of
# pointing this at a development one.
echo "Resolving organization..."
ORGANIZATION_ID="$(curl -fsS -m 30 "${ENDPOINT}/v1/teams" -b "$COOKIES" -c "$COOKIES" \
  -H 'Accept: application/json' -H 'X-Appwrite-Project: console' \
  | jq -r '.teams[0]."$id" // empty')"

if [ -z "$ORGANIZATION_ID" ]; then
  if ! ORGANIZATION_ID="$(api POST /teams '{"teamId":"unique()","name":"terraform-acceptance"}' \
    -H 'X-Appwrite-Project: console' | jq -r '."$id"')"; then
    echo "::error::Could not create an organization. A self-hosted instance allows only one, and a freshly signed-up account cannot join an existing one -- start from a clean instance." >&2
    exit 1
  fi
fi
require "$ORGANIZATION_ID" "the organization ID"

echo "Creating project..."
PROJECT_ID="$(api POST /organization/projects \
  '{"projectId":"unique()","name":"terraform-acceptance","region":"default"}' \
  -H 'X-Appwrite-Project: console' \
  -H "X-Appwrite-Organization: ${ORGANIZATION_ID}" | jq -r '."$id"')"
require "$PROJECT_ID" "the project ID"

# Read the scope list out of the running server rather than hardcoding it, so a
# version bump that adds a scope does not silently leave the key short of it.
SCOPES="$(docker exec "$CONTAINER" php -r \
  'echo json_encode(array_keys(require "/usr/src/code/app/config/scopes/project.php"));')"
require "$SCOPES" "the project scopes"

# Ephemeral keys are capped at one hour by the server, which comfortably clears
# the workflow's own timeout.
echo "Minting project API key..."
API_KEY="$(api POST /project/keys/ephemeral "$(jq -nc --argjson s "$SCOPES" \
  '{scopes:$s,duration:3600}')" \
  -H "X-Appwrite-Project: ${PROJECT_ID}" \
  -H 'X-Appwrite-Mode: admin' | jq -r '.secret')"
require "$API_KEY" "the API key"

# APPWRITE_ORGANIZATION_ID is deliberately not exported. Organization API keys
# are a Cloud-only feature -- self-hosted Appwrite does not route
# /v1/organization/keys at all -- and the provider rejects a project key on
# organization routes. Leaving it unset makes acceptance.OrganizationPreCheck
# skip those tests instead of failing them.
if [ -n "${GITHUB_ENV:-}" ]; then
  echo "::add-mask::${API_KEY}"
  {
    echo "APPWRITE_ENDPOINT=${ENDPOINT}/v1"
    echo "APPWRITE_PROJECT_ID=${PROJECT_ID}"
    echo "APPWRITE_API_KEY=${API_KEY}"
  } >> "$GITHUB_ENV"
  echo "Credentials written to \$GITHUB_ENV (project ${PROJECT_ID})"
else
  echo "export APPWRITE_ENDPOINT=${ENDPOINT}/v1"
  echo "export APPWRITE_PROJECT_ID=${PROJECT_ID}"
  echo "export APPWRITE_API_KEY=${API_KEY}"
fi
