# EnvDoctor

Configuration should have a contract.

EnvDoctor is an open-source, **local-first** configuration-contract CLI. It
defines the contract between an application and its environment, validates an
environment against that contract, and explains configuration failures without
exposing secrets.

> **Define your environment contract once, then validate any environment
> against it.**

EnvDoctor is:

- **language-agnostic** - it works with any application that reads environment
  variables;
- **local-first and offline** - normal validation makes no network requests;
- **read-only by default** - validation never writes files;
- **security-conscious** - secret values are never emitted anywhere;
- **deterministic** - identical environments produce byte-identical output on
  Linux, Windows, and macOS.

## Why

Configuration requirements are usually implicit and fragmented - scattered
across source code, `.env`, `.env.example`, CI, Docker files, and READMEs.
That fragmentation produces missing variables, typos, wrong types, stale
example files, and local/CI drift. EnvDoctor makes the requirements explicit as
a single contract (`envdoctor.schema.json`), then answers one question:

> **Does this environment satisfy the application's declared configuration
> contract?**

EnvDoctor is not a secrets manager, a dotenv manager, a deployment platform, a
cloud service, or a runtime configuration injector.

## Install

Requires Go 1.27 or newer. EnvDoctor is a single standalone binary with no
runtime dependencies.

```text
go install github.com/haneefojay/envdoctor/cmd/envdoctor@latest
```

Or build from source:

```text
git clone https://github.com/haneefojay/envdoctor
cd envdoctor
go build ./...
```

## Quickstart

Validate the process environment against the contract in the current directory:

```text
envdoctor check
```

Validate a dotenv file:

```text
envdoctor check --env-file .env
```

Machine-readable output for CI scripts:

```text
envdoctor check --env-file .env --json
```

Generate `.env.example` from the contract (never overwrites, never emits
secrets):

```text
envdoctor generate example
```

Bootstrap a draft contract from an existing `.env.example` or `.env` (copies
names only, never values):

```text
envdoctor init
```

Exit codes: `0` valid, `1` validation failure, `2` usage/contract/source error,
`3` unexpected internal error.

## Example

Contract `envdoctor.schema.json`:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["DATABASE_URL"],
  "properties": {
    "DATABASE_URL": {
      "type": "string",
      "format": "uri",
      "x-envdoctor-secret": true
    },
    "PORT": {
      "type": "integer",
      "minimum": 1,
      "maximum": 65535,
      "default": 3000
    },
    "LOG_LEVEL": {
      "type": "string",
      "enum": ["debug", "info", "warn", "error"],
      "default": "info"
    }
  }
}
```

`envdoctor check`:

```text
Environment validation failed

✗ DATABASE_URL
  ERROR ENV_MISSING
  missing required variable

1 error, 0 warnings
```

The contract's required variable is missing; `PORT` and `LOG_LEVEL` fall back to
their descriptive defaults (the application applies them, not EnvDoctor).

## Documentation

- `docs/quickstart.md` - get started in five minutes
- `docs/contract.md` - the contract language and semantics
- `docs/dotenv.md` - environment sources and the dotenv grammar
- `docs/diagnostics.md` - diagnostic codes, ordering, and exit codes
- `docs/security.md` - how secrets are protected
- `docs/ci.md` - CI matrix and local verification commands
- `docs/architecture.md` - package layout and invariants
- `docs/release.md` - semantic versioning, release pipeline, artifacts,
  checksums

## Status

MVP implementation of `envdoctor init`, `check`, and `generate example`. The
product requirements are governed by the specification, PRD, and roadmap under
`docs/01-consolidated-specification.md`, `docs/02-prd.md`, and
`docs/03-development-roadmap.md`; the implementation plan is
`IMPLEMENTATION_PLAN.md`. Behavior is locked by comprehensive unit and
end-to-end tests, including security and cross-platform suites.

## Security

Secret values are never emitted in output, diagnostics, or generated files.
See `SECURITY.md` and `docs/security.md`. Report vulnerabilities privately per
the policy in `SECURITY.md`.

## Development

```text
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

See `CONTRIBUTING.md` and `docs/ci.md`.

## License

MIT. See `LICENSE`.