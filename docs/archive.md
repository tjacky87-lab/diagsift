# Bundle and inspection format

A DiagSift bundle is an ordinary ZIP using safe relative forward-slash names. It
contains:

- `bundle.json`: format/version, manifest hash, counts, warnings, redaction counts,
  truncation flags, and SHA-256 hashes over final redacted entry bytes;
- `errors.json`: sanitized partial-failure records, only when failures occurred;
  collection records are hard-capped and the terminal record reports suppression;
- `REVIEW_BEFORE_SHARING.txt`: the mandatory local-review warning;
- `collectors/<collector-id>/...`: bounded redacted collector text.

Large entries that would exceed the compression-ratio limit are stored without
compression. Other entries use Deflate. Collection size ceilings still apply.

`inspect` is offline and never extracts entries. It rejects malformed ZIPs,
absolute/traversing/drive/colon/backslash names, duplicates and case collisions,
non-regular entries (including symlinks and devices), excessive entry or uncompressed sizes, extreme compression
ratios, inconsistent metadata/errors, and corrupt content hashes. It prints only
safe bundle-level metadata, never collector content.

Archive creation uses a private temporary file in the destination directory and
publishes it with an exclusive hard link after closing and syncing, then removes
the temporary name. Existing output paths, including dangling symlinks or names
created by another process during collection, are refused. The destination must
support hard links; unsupported filesystems fail closed. No overwrite-prone
rename or partial-copy fallback is used.
