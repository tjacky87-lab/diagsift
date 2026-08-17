# Security policy

DiagSift is pre-release software. Do not attach real secrets, user bundles,
production logs, or private host/account information to a public report.

When this repository is hosted, report suspected vulnerabilities through the
hosting provider's private security-advisory channel. Until a private reporting
channel is published, contact the maintainer privately rather than opening a
public issue. No response-time commitment is made before the first release.

## Scope and limitations

DiagSift is local-only and does not upload, add telemetry, or open issues. Its
redaction rules reduce exposure risk but cannot prove that content is safe.
Always inspect a bundle before sharing it. Command collectors use exact argv and
a restricted child environment, but allowed executables are not sandboxed.

Security fixes must use synthetic invalid canaries. Never add a real credential,
private log, user archive, personal path, email address, host, or account ID to a
test, fixture, issue, or commit.
