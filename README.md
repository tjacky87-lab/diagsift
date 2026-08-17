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

## Try it in five minutes

Prerequisite: one of the two currently supported Go release lines (Go 1.25 or
1.26; support window checked against go.dev on 2026-08-17).

```sh
go run ./cmd/diagsift validate examples/basic/diagsift.yaml
go run ./cmd/diagsift plan examples/basic/diagsift.yaml
go run ./cmd/diagsift collect examples/basic/diagsift.yaml --output basic.zip --yes
go run ./cmd/diagsift inspect basic.zip
```

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
