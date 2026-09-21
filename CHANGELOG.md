# Changelog

All notable changes to this project are documented in this file. The format is
based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## v0.1.0

### Added

- Project bootstrap (`go.mod`, CLI entry point, test infrastructure, CI,
  LICENSE, documentation skeleton).
- Contract model: `envdoctor.schema.json` parsing and validation as a
  restricted JSON Schema Draft 2020-12 profile, with stable contract
  diagnostics for unsupported keywords, duplicate definitions, version errors,
  and contradictory constraints.
- Environment sources: documented dotenv grammar (quoting, escapes, CRLF/LF,
  Unicode) and the process environment, normalized to the same internal
  representation. Duplicate dotenv keys and unsupported interpolation fail
  loudly instead of being silently resolved.
- Type normalization and validation: deterministic integer, number, and
  boolean grammar; string values preserved verbatim. `envdoctor check` reports
  stable structured diagnostics for every documented condition without ever
  disclosing secret values.
- Stable diagnostics and output: deterministic human and JSON reports
  (`{"valid", "diagnostics"}`), deterministic ordering, no ANSI sequences in
  JSON, and the documented exit codes (0 valid, 1 validation failure,
  2 usage/contract/source error, 3 unexpected internal error).
- `envdoctor check`: validate an explicit dotenv file (`--env-file`) or the
  current process environment (`--environment`) against
  `envdoctor.schema.json`, with `--json` for machine-readable output and
  contract discovery in the working directory.
- `envdoctor init`: create an initial reviewable contract from variable names
  discovered in `.env.example` or `.env`. It copies no values and infers no
  authoritative type, requiredness, secret classification, or default; every
  discovered variable is emitted as an optional string for review. Existing
  files are never silently overwritten.
- `envdoctor generate example`: generate `.env.example` from the contract.
  Only explicit non-secret defaults become values; secrets and variables
  without defaults are emitted blank; descriptions become comments; output is
  deterministic and existing files are never silently overwritten.
- Cross-platform invariant tests: human, JSON, and generated `.env.example`
  output are asserted to use LF line endings only, on every host OS.

### Security

- Consolidated secret non-disclosure suite running the real CLI: realistic
  secret values (super-secret-password, connection strings, token-like strings,
  values containing newlines and quotes) are forced through every value-driven
  diagnostic in human and JSON modes, plus malformed-file, init, and generate
  paths, and asserted never to reach stdout, stderr, JSON, or generated files.
- Static repo-wide guards (`internal/security`) that fail the build if shipping
  code imports or calls a banned capability: network access, TLS, shell
  execution, subprocess spawning, logging frameworks, telemetry, panic, and any
  environment-dump use outside the documented process source.

### Changed

- CI matrix extended: native build and test on Linux, Windows, and macOS
  runners (with the race detector on Linux and macOS, where a C toolchain is
  available), plus a cross-compile job covering all six targets
  (linux/windows/darwin x amd64/arm64). The gofmt check now runs under
  POSIX-shell semantics on every runner.

### Integration

- Realistic integration fixtures under `testdata/` (contracts, dotenv files,
  and a CI process environment) exercised end to end by a new
  `internal/integration` suite: every value-driven diagnostic, contract and
  source error classes, unknown-variable warnings, UTF-8 and LF/CRLF
  equivalence, and secret non-disclosure with deterministic output.

### Self-validation

- The repository now ships its own `envdoctor.schema.json`, and EnvDoctor
  validates it: `envdoctor check` in the repository validates the process
  environment by default (finding every repository variable optional, so the
  tool is green on developer machines) and the CI crossbuild matrix. The
  integration suite runs the matrix and the failure modes against the real
  contract file on every CI platform. Documented the dotenv BOM behavior
  surfaced by dogfooding (a leading UTF-8 BOM is rejected).

### Documentation

- New `docs/` reference: quickstart, contract, dotenv, diagnostics, security,
  CI, and architecture, written against the implemented behavior; README
  expanded with what/why/install/example/contract/security/documentation
  sections.

### Release engineering

- Semantic versioning is enforced: `envdoctor version` is now a
  `-ldflags`-injectable `var` (default kept in sync with the latest release so
  `go install ...@tag` reports a truthful version), guarded by a semver test.
- New `tools/release` packager builds all six documented targets with the
  version injected, wraps each binary with `README.md` and `LICENSE` in a
  deterministic (fixed member order and timestamps) `tar.gz`/`zip`, and writes
  a `SHA256SUMS` manifest plus a `NOTES.md` changelog excerpt.
- New `.github/workflows/release.yml`: tag-triggered pipeline that runs the
  full QA gate (gofmt, vet, tests), verifies the tag matches the embedded
  version, packages artifacts, and opens a draft GitHub Release.
- New `docs/release.md` documenting versioning policy, the compatibility
  surfaces (contract semantics, diagnostic codes, JSON output, exit codes),
  the release checklist, artifact layout, and checksum verification.