# Security and privacy model

DiagSift is a local collection tool, not a sandbox or anonymization product.

## Trust boundaries

- The manifest is untrusted and strictly decoded before any collector runs.
- File collectors accept only explicitly listed regular files confined to explicit
  roots and relative paths. Directories, globs, symlinks, junctions, and other
  reparse points are rejected or skipped in P0; no recursive traversal occurs.
- Command collectors use an exact executable plus argv. Shell interpreters are
  denied as executables and the child receives a minimal environment.
- Allowed executables are not sandboxed; they may use the user's normal filesystem
  and network permissions. Windows P0 guarantees direct-process timeout cleanup,
  while Unix uses a process group for descendant cleanup.
- Text is bounded in memory, redacted as a complete stream, and only then written
  to a private staging directory. Binary and invalid UTF-8 data is skipped.
- Recorded collector errors have a hard ceiling; the final bounded record reports
  that additional errors were suppressed.
- The final ZIP stays local. DiagSift has no upload, account, remote API,
  telemetry, analytics, crash upload, or update check.

## Limits

Redaction is pattern based. It can miss unknown formats, transform useful text,
or leave identifying context. Users must inspect every decompressed entry before
sharing. The portable P0 file-open path also has a small time-of-check/time-of-use
window between link checks and opening a file. These limitations must not be
described as guaranteed safety, privacy, compliance, or complete secret removal.

Tests use invalid synthetic canaries only and scan redaction output, staged text,
console/error/report channels, and every decompressed final ZIP entry.
