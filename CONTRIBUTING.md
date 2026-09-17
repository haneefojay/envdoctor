# Contributing

Read the specification and roadmap before changing behavior. Keep contract loading, source parsing, validation, diagnostics, and presentation separate. Prefer the Go standard library and deterministic output.

Before submitting a change:

```sh
gofmt -w .
go test ./...
go vet ./...
go build ./...
```

Review every change for secret leakage, nondeterminism, cross-platform behavior, and scope creep.
