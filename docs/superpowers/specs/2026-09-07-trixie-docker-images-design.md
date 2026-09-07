# Trixie Docker Images

## Problem

Several worker images use Debian 11 Bullseye. Bullseye LTS ended on August 31, 2026, and the final `bullseye-security` release metadata expired on September 7. Build stages that run `apt-get update` can no longer rely on a supported, current security repository, and the remaining Debian 11 runtime stages are unsupported.

## Design

Replace every explicit Debian 11 or Bullseye base with its Debian 13 Trixie equivalent while preserving language versions:

- Python build and runtime images
- TypeScript build and distroless runtime images
- Ruby build and runtime images
- Go and CLI distroless runtime images

Update the APT package constraints to versions supplied by Trixie. Keep packages constrained to their Trixie upstream versions while allowing Debian revision updates:

- Protobuf packages: `3.21.12-*`
- Clang: `1:19.0-*`

Do not disable APT expiration checks or redirect these images to a frozen Bullseye snapshot. Those approaches would conceal the unsupported operating-system dependency rather than remove it.

## Scope

Change only the five Dockerfiles containing explicit Debian 11 or Bullseye bases. Preserve the existing language major versions. Leave the Java Alpine/Jammy and .NET Jammy images unchanged because they are separate lifecycle and compatibility decisions. Keep this migration independent of the open feature stack.

## Verification

- Confirm every referenced Trixie and Debian 13 image tag exists for amd64 and arm64.
- Confirm the constrained package versions are available from Trixie.
- Build all affected worker images through the pull request's existing CI matrix.
- Confirm Dockerfile lint and repository formatting checks pass.
