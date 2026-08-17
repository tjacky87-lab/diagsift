# Exit codes and bundle contract

| Code | Meaning |
| ---: | --- |
| 0 | Command completed successfully (a collected bundle may still report non-fatal collector errors). |
| 2 | CLI usage error. |
| 3 | Manifest or policy validation failed. |
| 4 | Collection was not consented to. |
| 5 | Collection or bundle creation failed. |
| 6 | Bundle inspection rejected an unsafe, malformed, or corrupt archive. |

Bundles are ordinary ZIP files containing `bundle.json`,
`REVIEW_BEFORE_SHARING.txt`, redacted collector entries, and `errors.json` when a
collector partially fails. `bundle.json` records SHA-256 hashes over final redacted
entry bytes. Users must inspect every bundle and independently decide whether to
share it; redaction reduces risk but cannot guarantee safety.
