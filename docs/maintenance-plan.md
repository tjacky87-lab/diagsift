# Maintenance plan: September–December 2026

This is a proposed three-month work plan, not a promise of adoption or a support
service-level agreement. The project remains a small local diagnostic CLI.

| Period | Work | Reviewable outcome |
| --- | --- | --- |
| 8 Sep–7 Oct | Review pilot feedback; prioritize failed installation, unclear consent and missing diagnostic context. Use the two controlled cases as a baseline. | Public findings linked to relevant fixes, or an explicit record that no independent feedback arrived |
| 8 Oct–7 Nov | Address demonstrated blockers. Evaluate Windows descendant containment and race-resistant file opening with focused tests before making commitments. | Small reviewed PRs, regression evidence and documented remaining limits; ship a candidate only when changes justify one |
| 8 Nov–7 Dec | Revisit whether maintainers can supply manifests and users can review bundles independently. Assess the existing project pilot gate. | Evidence-based decision: stable release, another candidate, narrower scope, or a revised hypothesis |

## How Codex support would be used

- Review small proposed diffs, reproduce bugs and add synthetic regression cases.
- Triage public issues and connect a reported problem to its fix and validation.
- Check cross-platform behavior, update documentation and prepare releases.
- If API credits are requested, begin with bounded review jobs on public code and
  synthetic fixtures. Record usage, set a budget and check usefulness before
  extending automation. No API integration is required for the CLI itself.

Maintainer review and relevant checks remain the release gate. Changes suggested
by an agent need the same evidence as other changes. Private diagnostic bundles,
user logs and credentials will not be sent to an API as part of this plan.

## What to measure

Track completed independent exercises, where people get blocked, requests for
missing context, whether maintainers found a bundle useful, and repeat use when
reported. Distinguish first-time users, contributors and project maintainers.
Keep internal tests and verification downloads separate. Only report time saved
when there is a documented comparison; do not infer it from subprocess timings.

The existing two-project plus one non-maintainer target is a project validation
goal, not an OpenAI application requirement. Lack of feedback is a result to
report honestly, not a reason to invent activity or automatically broaden scope.

Uploads, telemetry, accounts, AI diagnosis, arbitrary shell execution, elevation
and broad filesystem discovery remain out of scope. Current limits are in the
[security model](security-model.md) and [release notes](../CHANGELOG.md).
