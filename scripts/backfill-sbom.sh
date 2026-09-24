#!/usr/bin/env bash
#
# Generates SBOMs for a release published before SBOMs were part of the release
# pipeline, and attaches them to that release.
#
#     GPG_FINGERPRINT=<release key> scripts/backfill-sbom.sh v1.8.0
#
# Deliberately a local maintainer script rather than a workflow. Signing needs
# the release GPG key, and a manually-dispatchable workflow holding that key
# would be a standing capability -- anyone able to dispatch workflows could
# reach the signing key at any time, for a job that runs once per old release
# and then never again. Release signing should only be reachable from a tag
# push. Run locally by someone who already holds the key and nothing new is
# exposed.
#
# The SBOMs are built by scanning the archives the release actually published,
# not by rebuilding the provider. Go embeds build metadata in the binary, so
# scanning the shipped artifact describes precisely what shipped; a rebuild
# today could resolve different dependencies and would be a different claim
# about a release already out in the world.
#
# The release's existing SHA256SUMS and its signature are never touched. The
# Terraform Registry recorded those checksums when the version was published and
# verifies downloads against them, so rewriting the file would invalidate the
# signature and risk breaking `terraform init` for anyone pinning that version.
# The backfilled SBOMs get their own checksum file, signed with the key this
# runs with. For a release published before a key rotation that is not the key
# which signed the release itself, which is why the two manifests are separate.
set -euo pipefail

TAG="${1:-}"
if [ -z "$TAG" ]; then
  echo "usage: $(basename "$0") <tag>    e.g. $(basename "$0") v1.8.0" >&2
  exit 64
fi

REPO="${REPO:-appwrite/terraform-provider-appwrite}"
VERSION="${TAG#v}"
SUMS="terraform-provider-appwrite_${VERSION}_SBOMS_SHA256SUMS"

# The release key is selected explicitly, exactly as the release workflow does.
# Falling back to gpg's default key would sign with whatever the maintainer
# happens to have first in their keyring, and consumers verifying against the
# documented release key would find the signature unauthenticatable.
GPG_FINGERPRINT="${GPG_FINGERPRINT:?GPG_FINGERPRINT must be set to the release signing key}"

for tool in gh syft gpg; do
  command -v "$tool" >/dev/null 2>&1 || { echo "error: $tool is required" >&2; exit 1; }
done

if ! gpg --list-secret-keys "$GPG_FINGERPRINT" >/dev/null 2>&1; then
  echo "error: no secret key for ${GPG_FINGERPRINT} in this keyring" >&2
  exit 1
fi

# sha256sum is GNU; macOS ships shasum instead.
if command -v sha256sum >/dev/null 2>&1; then
  sha256() { sha256sum "$@"; }
else
  sha256() { shasum -a 256 "$@"; }
fi

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
cd "$WORKDIR"

echo "Downloading ${TAG} archives from ${REPO}..."
gh release download "$TAG" --repo "$REPO" --pattern '*.zip'

for archive in *.zip; do
  echo "Cataloging ${archive}..."
  syft scan "file:${archive}" -o spdx-json > "${archive}.sbom.json"
done

sha256 ./*.sbom.json | sed 's| \./| |' > "$SUMS"
# No --batch here. The release workflow can use it because the GPG action
# preloads the passphrase into the agent; run locally against an ordinary
# keyring, --batch leaves gpg unable to ask for the passphrase and signing
# fails with "Inappropriate ioctl for device". This script is interactive
# anyway -- it prompts before uploading.
gpg --local-user "$GPG_FINGERPRINT" \
    --output "${SUMS}.sig" --detach-sign "$SUMS"

echo
echo "Generated $(find . -name '*.sbom.json' | wc -l | tr -d ' ') SBOMs for ${TAG}:"
cat "$SUMS"
echo
read -r -p "Upload these to the ${TAG} release on ${REPO}? [y/N] " reply
case "$reply" in
  [yY]) ;;
  *) echo "Aborted; nothing uploaded."; exit 0 ;;
esac

gh release upload "$TAG" --repo "$REPO" --clobber ./*.sbom.json "$SUMS" "${SUMS}.sig"
echo "Attached to ${TAG}."
