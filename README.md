# EnvDoctor

EnvDoctor is a local-first Go CLI for defining, validating, and diagnosing application configuration contracts.

## MVP commands

```sh
envdoctor init
envdoctor check --env-file .env
envdoctor check --environment --json
envdoctor generate example
```

The canonical contract is `envdoctor.schema.json`, using the restricted JSON Schema Draft 2020-12 profile documented in [`docs/contract.md`](docs/contract.md).

The MVP supports `string`, `integer`, `number`, and `boolean` variables; requiredness; enum and pattern constraints; numeric bounds; email and URI syntax checks; deterministic diagnostics; safe example generation; and warnings for unknown environment variables.

Secret values are never included in diagnostics, JSON, generated files, or logs. Normal validation performs no network requests and does not execute shell syntax.

## Development

```sh
gofmt -w .
go test ./...
go vet ./...
go build ./...
```

See [`docs/quickstart.md`](docs/quickstart.md), [`docs/architecture.md`](docs/architecture.md), [`SECURITY.md`](SECURITY.md), and [`CONTRIBUTING.md`](CONTRIBUTING.md).
