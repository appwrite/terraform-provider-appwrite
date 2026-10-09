# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in this Terraform provider, please report it responsibly.

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, please report vulnerabilities to [security@appwrite.io](mailto:security@appwrite.io).

Include the following information:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

We will acknowledge receipt within 48 hours and provide a detailed response within 7 days.

## Release Signing

Every release is signed with the project's release key: GoReleaser signs the
release's `SHA256SUMS`, and the Terraform Registry verifies provider downloads
against the same key.

| Releases                     | Key fingerprint                            |
| ---------------------------- | ------------------------------------------ |
| Up to and including `v2.1.0` | `20A01A01ED53E3472B0D98504C12BEF58D89E0C8` |
| After `v2.1.0`               | `9B826AE6CDD5C9AE9D3DCA4FEAF22BCD56186F6E` |

Both keys stay registered with the Terraform Registry. The older one is kept so
that releases already published continue to verify; it no longer signs anything
new.

```console
$ gpg --verify terraform-provider-appwrite_<version>_SHA256SUMS.sig \
               terraform-provider-appwrite_<version>_SHA256SUMS
```

## Software Bill of Materials

### Current releases

Releases from `v2.1.0` onwards publish an SBOM alongside each platform archive,
as `terraform-provider-appwrite_<version>_<os>_<arch>.zip.sbom.json`. They are
generated with [Syft](https://github.com/anchore/syft) in
[SPDX JSON](https://spdx.dev/) format and attached to the
[GitHub release](https://github.com/appwrite/terraform-provider-appwrite/releases)
next to the archive they describe.

Each SBOM is listed in the release's `SHA256SUMS`, which is signed with the
same GPG key as the rest of the release, so an SBOM can be verified before it
is trusted:

```console
$ gpg --verify terraform-provider-appwrite_<version>_SHA256SUMS.sig \
               terraform-provider-appwrite_<version>_SHA256SUMS
$ sha256sum --check --ignore-missing terraform-provider-appwrite_<version>_SHA256SUMS
```

To inspect what a release contains, for example to check it against a
vulnerability database:

```console
$ grype sbom:./terraform-provider-appwrite_<version>_linux_amd64.zip.sbom.json
```

### Releases published before SBOMs existed

Releases before `v2.1.0` do not carry SBOMs. They can be backfilled with
`scripts/backfill-sbom.sh`, which scans the archives a release originally
published rather than rebuilding it, so the result describes what actually
shipped instead of what a rebuild would produce today.

A backfilled release is verified differently from the ones above. A published
release's `SHA256SUMS` is recorded by the Terraform Registry when the version
goes out and is used to verify provider downloads, so it is never rewritten.
Backfilled SBOMs therefore carry their own checksum file, signed with the
current release key:

```console
$ gpg --verify terraform-provider-appwrite_<version>_SBOMS_SHA256SUMS.sig \
               terraform-provider-appwrite_<version>_SBOMS_SHA256SUMS
$ sha256sum --check --ignore-missing terraform-provider-appwrite_<version>_SBOMS_SHA256SUMS
```

Because the key that signs a backfill is the one current when the backfill runs,
it will not be the key that signed the release's own `SHA256SUMS` for any
release from before the key rotation. Verify an SBOM against the archive it
describes by comparing the archive checksum the SBOM records with the release's
own `SHA256SUMS` entry for that file.
