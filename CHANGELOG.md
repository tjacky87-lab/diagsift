# Changelog

## v0.1.0-rc.2 — 2026-09-08

- Store unusually compressible large entries without compression so valid local
  logs pass inspection without relaxing the compression-bomb limit.

- Redact quoted and short credential values and private-key blocks whose closing
  marker is missing, including collector-truncated input.
- Reject ZIP symlinks, devices, and other non-regular entries during inspection.
- Publish completed bundles without replacing existing or concurrently created
  output paths. Destination filesystems must support hard links.
- Enforce the manifest size limit while reading and reject non-regular manifests.
- Bound command output-pipe cleanup after exit or timeout on all platforms.
- Test Go 1.26 and 1.27, add binary installation instructions, and document a
  small independent pilot with explicit local content review.

All notable changes will be documented here. The project follows semantic
versioning once a first release is approved.

## Unreleased

- Store unusually compressible large entries without compression so valid local
  logs pass inspection without relaxing the compression-bomb limit.

## v0.1.0-rc.1 - 2026-08-18

- Initial local-first CLI contract with strict manifest validation and deterministic planning.
- Bounded explicit-file, OS/architecture, and exact-argv non-shell command collection.
- Pre-staging redaction with synthetic adversarial coverage.
- Ordinary ZIP creation, explicit consent, and hardened offline inspection.
- Cross-platform CI coverage for Windows, macOS, and Linux on Go 1.25 and 1.26.
- Release-candidate artifacts include six platform/architecture binaries, SPDX SBOM, SHA256 checksums, and build provenance.

This is a pre-release candidate. The generic-manifest adoption hypothesis still
requires independent maintainer/project pilots and a non-maintainer
bundle-generation/inspection exercise.
