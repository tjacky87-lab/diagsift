# A small independent pilot

The question is whether a maintainer's explicit manifest helps someone provide
useful troubleshooting context. A passing internal test is not an independent
pilot, and a download is not evidence that the exercise was completed.

## Choose one scenario

Use one real support problem you are authorized to investigate. For restic or
rclone, keep the scope to the program version, OS/architecture, and one named
log. Do not add config files, environment dumps, repository passwords, or entire
folders. The manifests in `examples/restic` and `examples/rclone` include only
that scope and start with a synthetic `diagnostic.log`.

The tool does not create application logs or diagnose the fault. Record the
expected behavior and actual symptom separately, omitting private paths and
account details. If this scope misses essential context, report what was
missing before expanding collection.

## Prepare and preview

1. Obtain and verify a release binary as described in the README, or build the
   source revision you intend to test. Record `diagsift version` and the revision.
2. Copy the relevant example directory into a dedicated local working folder.
   Read `diagsift.yaml`. The root `.` means that folder, not your current shell
   directory. Only `diagnostic.log` is listed; no folder is searched recursively.
3. Start with the synthetic log. If testing a real case, replace that copy with
   the one log you selected. Review its contents locally first.
4. From the working folder, run the following (on Windows use the full path to
   `diagsift.exe`, or `.\diagsift.exe` if it is in that folder):

```sh
diagsift validate diagsift.yaml
diagsift plan diagsift.yaml
diagsift collect diagsift.yaml --output pilot.zip
diagsift inspect pilot.zip
```

Read the resolved paths, executable/arguments, and limits before typing `YES`.
The version executable must be installed and available on PATH. Executables are
not sandboxed, even with the minimal environment and timeout. A missing program
or log is reported as a partial collection; do not count it as a complete pilot.

## Review and report

`inspect` checks the archive and content hashes; it does not show log contents or
prove privacy. Open the ZIP locally, read `bundle.json`, any `errors.json`, and
all collector entries. Look for truncation, missing context, and remaining
sensitive information. Leave the ZIP on your machine.

Report only non-sensitive observations in the
[pilot tracker](https://github.com/tjacky87-lab/diagsift/issues/4), or use the
**Pilot feedback** issue template. Include:

- Your role (maintainer, contributor, or first-time user), project, and OS.
- DiagSift version/revision and whether the log was synthetic or from a real case.
- Which of validate / plan / collect / inspect succeeded or blocked you.
- Whether collection matched the preview, and any errors or truncation.
- Whether the bundle helped explain the problem, including missing context.
- One improvement you would need before using the tool again.

Do not paste full plans, logs, credentials, or diagnostic ZIPs. Negative results
are useful. A first-time user should note where they needed maintainer help.
