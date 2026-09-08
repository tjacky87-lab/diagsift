# Small external review tasks

Choose one task. These are prepared invitations for voluntary review, not
completed contributions. No programming experience is needed for task A.

## A. Follow the Windows download instructions (about 10 minutes)

Start at the [README](../README.md), choose the correct rc.2 binary and compare
its checksum. Follow the [first-time-user exercise](https://github.com/tjacky87-lab/diagsift/issues/5)
using its synthetic log. Stop if you need help and note the exact step.

Useful feedback: OS/architecture, whether the right download was obvious,
whether checksum comparison was clear, and the first confusing sentence or error.
Do not paste personal paths, logs or the ZIP. Needing help is a valid result.

## B. Review one manifest (about 10 minutes)

Read either [restic](../examples/restic/diagsift.yaml) or
[rclone](../examples/rclone/diagsift.yaml). Without running it, describe what you
expect it to collect, its limits, and whether the version command and file root
are clear. Compare your expectation with `plan` if you choose to run it.

Useful feedback: anything collected that you did not expect, missing context
for one support problem, or a boundary you cannot understand. This is a manifest
review; it does not count as a completed real-project pilot.

## C. Review the bundle explanation (about 10 minutes)

Read the [recorded cases](evidence/cases.md) and [archive contract](archive.md).
Explain the difference between a hash-valid bundle and content that is safe to
share. Identify any wording that makes redaction sound guaranteed.

## How to report

Use the existing [pilot tracker](https://github.com/tjacky87-lab/diagsift/issues/4)
or the Pilot feedback template. State your role, selected task, DiagSift version,
whether you ran anything, and where assistance was needed. There is no need to
star the repository, endorse the tool, or provide positive feedback.
