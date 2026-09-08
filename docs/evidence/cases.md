# Two local support reproductions

Recorded on 8 September 2026 using DiagSift v0.1.0-rc.2, restic 0.19.1 and
rclone v1.75.1, on Linux amd64 (Ubuntu 24.04). These are maintainer-run fixtures
inspired by public support reports. They are not independent pilots, exact
recreations of the original users' environments, or upstream endorsements.

## 1. restic: a value passed where a password-file path is expected

The [public thread, post 11](https://forum.restic.net/t/fatal-wrong-password-or-no-key-found/3125/11)
shows confusion over `-p`. [Post 13](https://forum.restic.net/t/fatal-wrong-password-or-no-key-found/3125/13)
explains that this argument names a file. This case reproduces that specific
mistake, not the thread's separate Windows clipboard or escaping problems.
The [restic documentation](https://restic.readthedocs.io/en/stable/030_preparing_a_new_repo.html)
also describes password-file usage.

We initialized a disposable local repository with a fixture passphrase file.
Passing `-p not-a-password-file` to `snapshots` then failed with exit code 1:

```text
Fatal: Resolving password failed: Fatal: not-a-password-file does not exist
```

Passing the existing fixture file instead succeeded with exit code 0. No cloud
storage, real backup, user password, or original poster's data was used.

DiagSift collected the installed version, OS/architecture and one selected log.
The log retained the failed command context and missing-file error. This gives a
support reader a useful lead about argument interpretation. It cannot verify a
user's real password, recover a repository, or explain every authentication error.
The passphrase file and repository contents were outside the selected scope.

## 2. rclone: a remote name without its colon

In [this public report](https://forum.rclone.org/t/configuration-settings-return-error-error-listing-directory-not-found/19874),
a user tried listing remote names without a trailing colon. The maintainer
identified the missing separator. [rclone's local-backend documentation](https://rclone.org/local/)
explains local path syntax.

We created an alias remote named `demo` pointing to a new local directory with
one fixture file. `rclone --config fixture.conf ls demo -vv` failed with exit
code 3 and `directory not found`. Adding the colon (`ls demo:`) listed
`hello.txt` and returned exit code 0. This reproduces the name-parsing mistake
without OneDrive, encryption, or remote account access.

The selected log preserved the command, missing-directory error and rclone's
own warning about using `demo:`. **This modern rclone version already explains
the correction.** DiagSift's demonstrated role is packaging that context with
version and platform information. It did not discover the cause or improve
rclone's diagnosis. OS distribution/kernel, a separate symptom description and
backend intent may still be needed for a real support case.

## Observed results

| Check | restic | rclone |
| --- | --- | --- |
| Expected failing command | Exit 1 | Exit 3 |
| Corrected command | Exit 0 | Exit 0; fixture listed |
| validate / plan / collect / inspect | All exit 0 | All exit 0 |
| Collector entries | 3 | 3 |
| Collector errors / partial result | 0 / false | 0 / false |
| Hash verification | true | true |
| Selected error marker retained | Yes | Yes |

Each collection used the existing example manifest unchanged. The three entries
are version output, OS/architecture JSON and `diagnostic.log`. Inspector entry
count is 4 because its count includes the review-warning entry; it is not four
collectors. See [the complete record](results.json) for exact output and timings.
Timings are automated subprocess durations, not human usability measurements.

The fixture working-directory prefix was replaced with `<WORK>` before log
collection and in the published transcript. This normalization is performed by
the reproduction script; it is **not evidence of DiagSift path redaction**.
No real credentials were placed in the logs, so these cases do not test secret
removal. `inspect` verifies structure and hashes, not whether sharing is safe.

## Reproduce

Requirements: Python 3.10+, verified DiagSift/restic/rclone binaries, and a local
filesystem supporting hard links. The script was tested on Linux; other OSes
have not been exercised for this script. Obtain upstream binaries from their
official releases and verify their checksums. The script does not download tools.

From the DiagSift checkout:

```sh
python3 scripts/run-evidence-cases.py \
  --diagsift /absolute/path/to/diagsift \
  --restic /absolute/path/to/restic \
  --rclone /absolute/path/to/rclone \
  --output /absolute/path/to/new-case-run
```

Use a new output path. The script refuses an existing directory, supplies a
minimal environment and uses only disposable local fixtures. It records `YES`
as automated consent after recording each plan. The fixture setup and correction
commands run outside DiagSift; DiagSift only executes the version collectors.
Generated ZIPs remain in the output directory for local review.
