# Contributing

DiagSift is currently developed as a small, standard-library-first Go CLI. Before
submitting a change, run:

```sh
gofmt -w .
go test ./...
go vet ./...
git diff --check
```

Add focused tests with behavior changes, including negative/adversarial cases for
paths, archives, commands, limits, and redaction. Fixtures must be synthetic.
Never include real logs, bundles, credentials, personal data, or private hosts.

Do not weaken a safety gate to make a test pass. Public claims must remain
conservative, and no activity should be manufactured to imply adoption.
