# 80-second demo transcript

Actual captured CLI output, replayed with edited pauses and excerpts.
This is a maintainer-run fixture exercise, not an independent user recording.

## 01 / Verify the release

Real CLI output. Local fixtures. Pauses edited for readability.

```text
$ diagsift version
diagsift v0.1.0-rc.2
[exit 0]

Recorded binary SHA-256:
33c62f323a476ef39c3bdcc2450395855690b72be58c38cbf0b2cb1c3b59a896

Matched the published rc.2 SHA256SUMS.
Download: GitHub > tjacky87-lab/diagsift > Releases
```

## 02 / Reproduce a restic mistake

The -p argument needs a file path. The fixture file fixes this case.

```text
$ restic -r <WORK>/restic/repository -p not-a-password-file snapshots
Fatal: Resolving password failed: Fatal: not-a-password-file does not exist
[exit 1]

$ restic -r <WORK>/restic/repository -p <WORK>/restic/fixture-passphrase.txt snapshots

[exit 0]
```

## 03 / Reproduce a rclone mistake

The remote name needs a colon. rclone itself already explains this.

```text
$ rclone --config <WORK>/rclone/fixture.conf ls demo -vv
[earlier output omitted]
2026/09/08 10:50:12 NOTICE: "demo" refers to a local folder, use "demo:" to refer to your remote or "./demo" to hide this warning
2026/09/08 10:50:12 ERROR : error listing: directory not found
2026/09/08 10:50:12 DEBUG : 3 go routines active
2026/09/08 10:50:12 NOTICE: Failed to ls with 2 errors: last error was: directory not found
[exit 3]

$ rclone --config <WORK>/rclone/fixture.conf ls demo:
26 hello.txt
[exit 0]
```

## 04 / Validate and preview

The existing manifest collects version, OS/architecture and one log.

```text
$ diagsift validate <WORK>/rclone/selected/diagsift.yaml
valid: rclone-support (diagsift.yaml)
[exit 0]

$ diagsift plan <WORK>/rclone/selected/diagsift.yaml
[plan excerpt]
{
  "name": "rclone-support",
  "limits": {
    "maxFiles": 4,
    "maxTotalBytes": 2097152,
    "maxDuration": "15s"
  }
}
[collector summary: rclone version; os + arch; diagnostic.log]
```

## 05 / Explicit consent, local ZIP

The fixture runner supplies YES. This is not an independent user test.

```text
$ diagsift collect <WORK>/rclone/selected/diagsift.yaml --output <WORK>/rclone/selected/case.zip
[automated stdin: YES]
[earlier output omitted]
}
Type YES to collect this plan locally: created case.zip: 3 collector entries, 0 errors, partial=false
[exit 0]
```

## 06 / Inspect the archive

Checks structure and hashes. Does not certify safe-to-share content.

```text
$ diagsift inspect <WORK>/rclone/selected/case.zip
{
  "formatVersion": "diagsift.bundle/v1alpha1",
  "name": "rclone-support",
  "createdAt": "2026-09-08T02:50:12Z",
  "partial": false,
  "entryCount": 4,
  "errorCount": 0,
  "totalUncompressed": 2335,
  "hashesVerified": true
}
[exit 0]
```

## 07 / Read the actual collected log

Selected ZIP entry excerpt; disposable paths normalized before collection.

```text
collectors/log/diagnostic.log

Local fixture. Failing command: rclone --config fixture.conf ls demo -vv
2026/09/08 10:50:12 NOTICE: "demo" refers to a local folder, use "demo:" to refer to your remote or "./demo" to hide this warning
2026/09/08 10:50:12 ERROR : error listing: directory not found
2026/09/08 10:50:12 NOTICE: Failed to ls with 2 errors: last error was: directory not found
```

## 08 / What this evidence means

Full commands, outputs, source links and limitations are in the repository.

```text
Two controlled cases passed.

Each bundle: 3 collectors, 0 errors, partial=false.
Useful error text survived collection.
Corrected upstream commands succeeded.

Not demonstrated: independent adoption, time saved,
complete secret removal, or automatic diagnosis.

Read: docs/evidence/README.md
Try: GitHub issue #5 | Feedback: issue #4
```

## Rebuild the replay

Requires Python, Pillow, ffmpeg and a monospace TrueType font.

```sh
python3 scripts/render-evidence-demo.py --record docs/evidence/results.json \
  --output /absolute/path/to/new-demo-output \
  --font /absolute/path/to/DejaVuSansMono.ttf
```
