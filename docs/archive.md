# Bundle and inspection format

A DiagSift bundle is an ordinary ZIP using safe relative forward-slash names. It
contains:

- `bundle.json`: format/version, manifest hash, counts, warnings, redaction counts,
  truncation flags, and SHA-256 hashes over final redacted entry bytes;
- `errors.json`: sanitized partial-failure records, only when failures occurred;
  collection records are hard-capped and the terminal record reports suppression;
- `REVIEW_BEFORE_SHARING.txt`: the mandatory local-review warning;
- `collectors/<collector-id>/...`: bounded redacted collector text.

`inspect` is offline and never extracts entries. It rejects malformed ZIPs,
absolute/traversing/drive/colon/backslash names, duplicates and case collisions,
unexpected entries, excessive entry or uncompressed sizes, extreme compression
ratios, inconsistent metadata/errors, and corrupt content hashes. It prints only
safe bundle-level metadata, never collector content.

Archive creation uses a private temporary file in the destination directory and
renames it into place after closing and syncing. Existing output paths are
refused. Atomicity ultimately follows the destination filesystem's rename
semantics.
