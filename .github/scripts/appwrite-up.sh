#!/usr/bin/env bash
#
# Boots a self-hosted Appwrite for the acceptance tests to run against.
#
# The compose file and its .env are downloaded from the pinned upstream tag
# rather than vendored here: they are ~2000 lines that have to stay consistent
# with the server image, and pinning the tag makes them immutable anyway.
#
# Only a subset of the ~30 upstream services is started. Compose pulls in
# whatever those depend on (the database, redis, geo, autogravity), which is
# enough to serve the API the provider talks to. Starting the full stack would
# add clickhouse, mongo, a headless browser and a dozen idle workers for no gain.
set -euo pipefail

APPWRITE_VERSION="${APPWRITE_VERSION:?APPWRITE_VERSION must be set}"
WORKDIR="${APPWRITE_WORKDIR:-/tmp/appwrite-ci}"
BASE_URL="https://raw.githubusercontent.com/appwrite/appwrite/${APPWRITE_VERSION}"

# The API plus the workers the acceptance tests actually wait on. Resources are
# created synchronously, but deletes and column/index creation are queued, so
# without these two the teardown hangs and the index tests time out.
#
# The build workers, orchestrator and runtime executor are deliberately absent:
# only the deployment-build tests need them, those are gated behind
# APPWRITE_BUILD_TESTS, and running them here costs enough memory to get the
# API container OOM-killed.
SERVICES=(
  traefik
  appwrite
  appwrite-worker-databases
  appwrite-worker-deletes
)

mkdir -p "$WORKDIR"
curl -fsSL "${BASE_URL}/docker-compose.yml" -o "${WORKDIR}/docker-compose.yml"
curl -fsSL "${BASE_URL}/.env" -o "${WORKDIR}/.env"

# The compose file defaults the image tag to :latest. A nightly job must not
# start failing because upstream published a release overnight.
export _APP_VERSION="$APPWRITE_VERSION"

docker compose --project-directory "$WORKDIR" up -d "${SERVICES[@]}"

echo "Waiting for Appwrite to become healthy..."
for _ in $(seq 1 60); do
  if curl -fsS -m 5 -o /dev/null "http://localhost/v1/health/version" 2>/dev/null; then
    echo "Appwrite ${APPWRITE_VERSION} is up: $(curl -fsS -m 5 http://localhost/v1/health/version)"
    exit 0
  fi
  sleep 5
done

echo "::error::Appwrite did not become healthy within 5 minutes"
docker compose --project-directory "$WORKDIR" ps
exit 1
