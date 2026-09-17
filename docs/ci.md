# CI Usage

Run the local verification script from the repository root:

```sh
sh scripts/ci.sh
```

It formats the Go sources, verifies formatting, runs unit tests, runs `go vet`, and builds the CLI. The intended GitHub Actions workflow uses the same checks; adding workflow files requires repository workflow-write permission.

For application validation in CI:

```sh
envdoctor check --environment --json
```
