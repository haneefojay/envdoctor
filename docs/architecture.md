# EnvDoctor Architecture

This document describes how EnvDoctor is structured and the invariants that
hold across every level. It describes the implementation as it exists today
(the MVP); the governing product behavior lives in the consolidated
specification (`docs/01-consolidated-specification.md`).

## Core principle

EnvDoctor treats configuration as a contract. Validation is always a pure
function of two inputs:

```text
contract + environment -> diagnostics
```

The core validation logic does not depend on where the environment came from:
a dotenv file, the process environment, or any future source. The contract is
the source of truth, the environment is evaluated against it, and the result is
a deterministic set of structured diagnostics.

## Data flow

```text
Configuration Source
        |
        v
Source Parser
        |
        v
Normalized Environment (map[string]string)
        |
        v
Contract Validator
        |
        v
Diagnostics
        |
        v
Human / JSON Output
```

The contract file (`envdoctor.schema.json`) is parsed on a parallel path and
also feeds the validator. Output never reads the environment directly; it
consumes only the diagnostics, which is what makes secret non-disclosure
architecturally enforced rather than merely careful.

## Package layout

| Package               | Responsibility                                                         |
| --------------------- | ---------------------------------------------------------------------- |
| `cmd/envdoctor`       | Command-line interface: `init`, `check`, `generate example`, version, help. Owns flag parsing, file paths, exit codes. |
| `internal/contract`   | The EnvDoctor Contract Profile: parses and normalizes the restricted JSON Schema Draft 2020-12 document into a `Contract`. Rejects unsupported constructs with contract diagnostics. |
| `internal/source`     | Environment sources. `Process`/`FromProcess` read the process environment; the dotenv parser reads and validates dotenv files. Both produce `map[string]string`. |
| `internal/normalize`  | The fixed value grammar: interprets environment strings into `string`, `int64`, `float64`, or `bool` values. Never trims or coerces silently. |
| `internal/validate`   | The validation engine: `Validate(contract, environment) -> diagnostics`. Presentation- and source-agnostic. |
| `internal/diagnostic` | The stable structured diagnostic shape and stable codes. |
| `internal/output`     | Deterministic human and JSON rendering of a `Result`. |
| `internal/generate`   | Renders the deterministic `.env.example` from contract metadata only. |
| `internal/init`       | Adoption assistant: discovers variable names only and emits a reviewable draft contract. |
| `internal/security`   | Static, repo-wide guards that keep forbidden capabilities out of shipping code. |

There is no speculative layering beyond these packages; each has a single,
clear boundary.

## Architectural rules

- **Separation of concerns.** Parsing, validation, diagnostics, and
  presentation live in separate packages. The validator knows nothing about
  terminals or dotenv syntax; the output layer knows nothing about the
  environment.
- **Determinism.** No behavior depends on Go map iteration order. Diagnostics
  are canonicalized (see `docs/diagnostics.md`), contract variables are sorted
  by name, and output is byte-identical for identical input on every platform.
- **One source per invocation.** MVP sources are never merged and have no
  precedence rules. `envdoctor check` validates exactly one explicitly selected
  source: a dotenv file or the process environment (the default).
- **Secrets boundary.** Environment values are treated as toxic. Diagnostics
  are the safety boundary: neither the validator, the diagnostics, nor the
  output layer ever embeds a value in a message. There is no environment-dump
  or "verbose secrets" mode. See `docs/security.md`.
- **Read-only by default.** Validation performs no network access, no DNS or
  service checks, no shell execution, no subprocesses, and no writes. Only
  `generate` and `init` write files, and both refuse to overwrite an existing
  file.
- **Standard library only.** EnvDoctor builds as a standalone executable with
  no runtime dependency. Third-party dependencies are avoided unless they
  materially improve correctness or remove significant complexity.

## Cross-platform posture

- Output and generated files use LF line endings on every host operating
  system; there is no raw carriage-return byte in human, JSON, or generated
  output.
- The dotenv parser accepts both LF and CRLF line endings deterministically and
  preserves UTF-8 byte-for-byte.
- EnvDoctor never emits ANSI escape sequences and therefore needs no terminal
  detection.
- Environment variable names are semantically case-sensitive regardless of the
  host operating system: names are never case-normalized.

## Security enforcement

`internal/security` is a test-only package that scans all non-test Go code under
`cmd/` and `internal/` and fails the build if a banned capability appears:
network clients, TLS, raw dialing, `os/exec`, the `log`/`slog` frameworks,
`panic(`, telemetry markers, or `os.Environ()` anywhere except the documented
process source. It is the belt-and-suspenders layer under the behavior-level
secret tests in `cmd/envdoctor`.