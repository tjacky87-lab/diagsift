# Manifest contract (`diagsift/v1alpha1`)

DiagSift manifests are local YAML files decoded with unknown-field rejection. The
top-level `apiVersion` and `kind` are fixed. Every file collector names an explicit
root and one or more relative paths to regular files beneath it. Directories and
globs are rejected in v0.1; directory/glob collection is deferred pending pilot
evidence. Absolute collector paths, traversal, archive traversal, duplicate IDs,
unsupported fields, and limits above compiled hard ceilings are rejected before
collection.

The published JSON Schema provides structural and editor validation. `diagsift
validate` is authoritative for semantic and security-policy validation, including
duration ceilings, duplicate IDs, cross-references, aggregate limits, and command
policy. Complex Go-duration ceiling semantics are intentionally not duplicated in
fragile schema regular expressions.

Command collectors specify an exact executable and argument array. DiagSift never
interpolates environment variables. The shell interpreters `cmd`, `powershell`,
`pwsh`, `sh`, `bash`, `zsh`, `dash`, and `ksh` are rejected as executables,
including path-qualified and common executable-suffix forms. Any executable whose
final filename ends in `.bat` or `.cmd` is also rejected. This is not a sandbox:
an allowed executable can still read files, access the network, or cause other
effects permitted to the user.

The P0 system collector exposes only `os` and `arch`.

Current compiled ceilings are 1,000 files, 64 MiB total content, 8 MiB per file,
4 MiB per command stream, 5 minutes overall, 1 minute per command, 100 collectors,
and 500 explicitly listed file paths. Recorded collector errors are capped at 256,
with the final record indicating that additional errors were suppressed. A manifest
can only lower configurable ceilings.
