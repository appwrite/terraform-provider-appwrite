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

## Software Bill of Materials

Every release publishes an SBOM alongside each platform archive, as
`terraform-provider-appwrite_<version>_<os>_<arch>.zip.sbom.json`. They are
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
