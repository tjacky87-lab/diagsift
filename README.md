# DiagSift

DiagSift is an experimental local-first CLI for open-source maintainers who need
a bounded, reviewable diagnostic bundle from a user's machine. A maintainer writes
a small `diagsift.yaml`; the user validates and previews it, explicitly consents,
creates an ordinary ZIP locally, inspects that ZIP, and independently decides
whether to share it.

It addresses a common support gap between vague requests for "all the logs" and
large product-specific diagnostic systems. HashiCorp hcdiag and Replicated
Troubleshoot provide mature diagnostics in their ecosystems, while sos targets
broader system reporting. DiagSift's narrower hypothesis is that independent
projects may benefit from one project-neutral, cross-platform manifest contract.
That adoption hypothesis is not yet proven.

DiagSift never uploads bundles, opens issues, calls a remote API, adds telemetry,
or permits shell interpreters as command collectors. Redaction reduces risk but does **not** guarantee a
bundle is safe to share. Allowed child executables are **not sandboxed** and can
still perform actions permitted to the user.

## Get DiagSift

Prebuilt Windows, macOS, and Linux binaries (amd64 and arm64) are on the
[release page](https://github.com/tjacky87-lab/diagsift/releases/tag/v0.1.0-rc.1).
Download the binary for your OS/architecture and `SHA256SUMS` from the same release.
Compare the file's SHA-256 with its matching line before running it:

```powershell
Get-FileHash .\diagsift-v0.1.0-rc.1-windows-amd64.exe -Algorithm SHA256
```

On Linux, use `sha256sum <downloaded-file>`; on macOS, use
`shasum -a 256 <downloaded-file>`. Then rename the verified binary to
`diagsift.exe` on Windows or `diagsift` on macOS/Linux. On macOS/Linux, run
`chmod +x ./diagsift`. These binaries do not require Go.

Published rc.1 binaries are experimental. Changes described under **Unreleased**
in [CHANGELOG.md](CHANGELOG.md) require a source build until a new release exists.

## Try it in five minutes

The [first-time-user pilot](https://github.com/tjacky87-lab/diagsift/issues/5)
includes a synthetic example and step-by-step instructions. For a real project,
follow the [pilot guide](docs/pilot-guide.md). Neither exercise requires sharing
a diagnostic bundle.

To build from source, use Go 1.26 or 1.27 (the CI-tested release lines; checked
2026-09-08 against [go.dev](https://go.dev/dl/)):

```sh
git clone https://github.com/tjacky87-lab/diagsift.git
cd diagsift
go run ./cmd/diagsift validate examples/basic/diagsift.yaml
go run ./cmd/diagsift plan examples/basic/diagsift.yaml
go run ./cmd/diagsift collect examples/basic/diagsift.yaml --output basic.zip
go run ./cmd/diagsift inspect basic.zip
```

Collection displays the plan and asks you to type `YES`. The `--yes` option is
for automation after separately reviewing the plan. Use a new output filename
for each run; existing paths are never overwritten. Save on a filesystem that
supports hard links (for example NTFS, APFS, or ext4). Unsupported destinations
fail without replacing existing data.

`inspect` verifies structure, limits, and hashes. **It does not display the log
contents or certify that secrets are absent.** Open the ZIP in a local archive
viewer, read every entry, and decide independently whether to share anything.

## Safety model

- Manifests fail closed on unknown fields, versions, duplicate IDs, unsafe paths,
  shell interpreters, and limits above compiled ceilings.
- Every file path is an explicitly listed regular file relative to an explicit
  collection root. Directories and globs are rejected in v0.1.
- Planning is deterministic and does not execute collectors or subprocesses.
- Collection is bounded by time, file-count, per-entry, and total-size limits.
- Captured text passes through configured redaction before durable bundle content.
- Bundles stay local and require explicit review before sharing.

Read [the manifest contract](docs/manifest.md), [security model](docs/security-model.md),
[archive format](docs/archive.md), [exit codes and bundle contract](docs/exit-codes.md),
and [the security policy](SECURITY.md) before creating a real manifest.

## Status

DiagSift is pre-release software. The generic-manifest adoption hypothesis still
requires two independent maintainer/project pilots and one non-maintainer bundle
generation/inspection exercise. Do not use it as evidence of guaranteed
anonymization, privacy, compliance, or complete secret removal.

## License

Apache License 2.0. See [LICENSE](LICENSE).
