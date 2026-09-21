# Contributing

Thanks for considering a contribution to EnvDoctor.

## Governing Documents

Before contributing, read, in order:

1. `docs/01-consolidated-specification.md`
2. `docs/02-prd.md`
3. `docs/03-development-roadmap.md`
4. `AGENTS.md`

The specification is authoritative. Do not invent requirements that contradict
it.

## Development

Requirements: Go 1.27+.

```text
go build ./...
go test ./...
go vet ./...
```

Run formatting with `gofmt -l -w .` and format check `test -z "$(gofmt -l .)"`.

## Conventions

- Prefer the Go standard library. Add dependencies only when they materially
  improve correctness or remove significant complexity.
- Prefer table-driven tests.
- Output must be deterministic and secrets must never be emitted.
- Keep parsing, validation, diagnostics, and presentation separate.
- No scope creep: implement the phase described by the roadmap.
- Do not rename existing public identifiers casually; diagnostic codes, JSON
  shape, and exit codes are compatibility surfaces.

## Pull Requests

- Keep changes small and focused on one roadmap phase.
- Include tests for new behavior.
- Run `go build ./...`, `go vet ./...`, and `go test ./...` before opening.
- Ensure gofmt is clean.

## License

By contributing you agree that your contribution is licensed under the MIT
License (see `LICENSE`).