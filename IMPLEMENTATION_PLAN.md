# EnvDoctor — Implementation Plan

**Baseline:** docs/01-consolidated-specification.md, docs/02-prd.md,
docs/03-development-roadmap.md, AGENTS.md, CLAUDE.md
**Language:** Go (standard library only)
**Status:** P0–P16 complete (check, generate example, init, security hardening,
cross-platform hardening, documentation, integration fixtures, self-validation,
release engineering); P17 pending.

## Guiding Constraints

- Contract is the source of truth; `validate(contract, environment) -> diagnostics`.
- Deterministic everywhere: never rely on Go map iteration order.
- Secrets never appear in stdout, stderr, JSON, diagnostics, errors, generated
  files, snapshots, or test output.
- Cross-platform: Linux/Windows/macOS, UTF-8, LF/CRLF, path behavior.
- No scope creep: no network, no shell execution, no source merging, no
  speculative packages.
- Definition of Done per phase: code + focused tests + docs where behavior
  changes + security/determinism/cross-platform review.

## Current Repository State

Implemented:
- P0 base: go.mod (module github.com/haneefojay/envdoctor, go 1.27.1),
  cmd/envdoctor placeholder (version/help only), Makefile, CI matrix
  (ubuntu/windows/macos), LICENSE, SECURITY.md, CONTRIBUTING.md, CHANGELOG.md,
  README skeleton.
- P1 contract model: internal/contract — implemented with table-driven tests.
- P2 environment sources: internal/source (process + dotenv) — implemented with
  tests.
- P3 type normalization: internal/normalize — implemented with tests.
- P4 validation engine: internal/validate — implemented with tests.
- P5 diagnostic model: internal/diagnostic — implemented with tests.
- P6/P7 output: internal/output (Result/NewResult, WriteHuman, WriteJSON) —
  implemented with golden tests (success, one error, many errors, warning-only,
  secret failure, malformed source, malformed contract).
- P8 CLI: cmd/envdoctor — run() over io.Writer, `check` implemented end-to-end
  (--contract, --env-file, --environment, --json; contract discovery of
  envdoctor.schema.json; stable exit codes 0/1/2/3) with end-to-end tests.
  `init` and `generate example` are wired (see P9/P10).
- P9 example generator: internal/generate — `Example` renders the deterministic
  .env.example from contract metadata only (default -> value, secret/enum/
  pattern/type without default -> blank, descriptions -> comments, round-trip
  safe string escaping); `WriteExample` refuses overwrite atomically (O_EXCL);
  `generate example` CLI (--contract, --output, exit codes 0/2) with tests
  covering value rules, secret non-disclosure, no-overwrite, determinism, and
  dotenv round-trips.
- P10 init: internal/init — `Discover` returns only variable names from
  .env.example (preferred) or .env, copying no values; `Template` emits every
  discovered variable as an optional string with an advisory description;
  `WriteSchema` refuses overwrite atomically (O_EXCL); `init` CLI
  (--output, exit codes 0/2) with tests covering name-only discovery,
  deduplication/sorting, determinism, neutrality, and non-overwrite.
- P11 security hardening: cmd/envdoctor/security_test.go drives realistic
  secret values (super-secret-password, postgres://user:password@host/db,
  token-like strings, newlines, quotes) through every value-driven diagnostic
  in both human and JSON modes plus malformed-file, init, and generate paths,
  asserting none reach stdout/stderr/generated files; internal/security keeps
  a static guard test that bans network, TLS, shell, logging, telemetry, panic,
  and environment-dump capability from non-test code.
- P12 cross-platform hardening: CI builds and tests natively on
  ubuntu/windows/macos (race detector on Linux/macOS, where a C toolchain
  exists) and cross-compiles all six targets (linux/windows/darwin x
  amd64/arm64) with CGO disabled; the gofmt check is portable across shells;
  LF-only line-ending invariants are locked by tests for human, JSON, and
  generated .env.example output.
- P13 documentation: docs/quickstart.md, docs/contract.md, docs/dotenv.md,
  docs/diagnostics.md, docs/security.md, docs/ci.md, docs/architecture.md
  written against implemented behavior; README rewritten with what/why/
  install/example/contract/CI/security/documentation links.
- P14 integration fixtures: testdata/ populated with contracts/ (FastAPI,
  NestJS, Node API, frontend NEXT_PUBLIC_*, secret-heavy backend,
  invalid-values, unknown-vars, plus malformed/unsupported-keyword/
  duplicate-variable error contracts), dotenv/ (valid + broken envs, malformed
  dotenv, UTF-8/Unicode), and environments/ (CI process env as raw KEY=VALUE).
  internal/integration runs the full pipeline over the fixtures: every value
  diagnostic is exercised, contract and source error classes are asserted,
  unknown-variable warnings and warning-does-not-invalidate behavior are
  locked, UTF-8 + LF/CRLF equivalence is proven, and realistic secrets are
  verified absent from human/JSON output with byte-identical determinism.

- P15 self-validation: envdoctor.schema.json added at the repository root,
  contracting the only variables the repo actually sets as environment
  configuration (the CI crossbuild job's GOOS/GOARCH/CGO_ENABLED); all
  optional, so plain `envdoctor check` in the repo is always green while
  bogus build settings fail with usable diagnostics. internal/integration
  keeps the repo contract valid on the CI matrix (all six GOOS/GOARCH
  combos validate; fuchsia/x86_64/yes/2 rejected with the documented codes)
  and dogfooding documented: Windows editors must save dotenv files without a
  BOM, since a leading BOM is rejected deterministically.

- P16 release engineering: semantic versioning enforced end to end. `version`
  is now a var injectable via `-X main.version` (default kept in sync so
  `go install @tag` reports truthfully; a semver test guards it). The
  `tools/release` packager (deliberately under tools/ so the cmd/ and
  internal/ security guards stay undisturbed by its `os/exec` need) builds all
  six targets with CGO disabled, wraps binary+README+LICENSE in deterministic
  tar.gz/zip archives, and writes SHA256SUMS plus a NOTES.md changelog excerpt;
  a tag-triggered GitHub Actions workflow runs gofmt/vet/tests, verifies the
  tag matches the embedded version, packages, and opens a draft release.
  docs/release.md documents versioning, the compatibility surfaces, and
  verification. Locally validated: native + cross artifacts, checksums, and
  `-X` version embedding proven by executing a packaged binary.

Gaps: none outstanding for P0–P16.

## Phase Plan

### P1 gap close — Contract tests
internal/contract/*_test.go, table-driven:
- valid contracts load into normalized Contract;
- unsupported root/variable keywords -> CONTRACT_UNSUPPORTED_KEYWORD;
- malformed JSON, trailing content, duplicate keys -> CONTRACT_INVALID /
  CONTRACT_DUPLICATE_VARIABLE;
- unsupported $schema / missing $schema -> CONTRACT_INVALID;
- root type must be object; required referencing undeclared variable -> error;
- required+default and secret+default -> CONTRACT_INVALID;
- enum/const type interpretation; enum empty; enum entries invalid for type;
- pattern compile failure; minLength>maxLength; minimum>maximum; exclusive
  bound contradictions; const not in enum;
- numeric/string constraints on wrong type;
- defaults violating their constraints;
- deterministic diagnostic ordering.
Exit: go test ./internal/contract passes with full coverage of P1 cases.

### P2 — Environment sources (internal/source)
- ProcessEnv: reads os.Environ(), preserves names/values exactly (no
  case-normalization, no trimming) -> map[string]string.
- Dotenv parser: explicit grammar. KEY=value, blank lines, comments,
  single-quoted literals, double-quoted values with documented escapes, empty
  values, LF and CRLF, UTF-8.
- Parser must reject: duplicate keys (no silent resolution), interpolation or
  command substitution / shell syntax -> clear source error (ENV_FILE_INVALID /
  SOURCE_INVALID), never evaluate.
- Malformed input reported as env-file error, distinguishable from contract
  errors.
Tests: comments, blank lines, quoting, empty values, malformed lines, duplicate
keys, unsupported interpolation, CRLF, LF, Unicode; process-env fidelity.
Exit: both sources normalize to identical map[string]string; validator remains
source-agnostic.

### P3 — Type normalization (internal/normalize)
- string: passthrough, no trimming.
- integer: accept 0, 1, 3000, -10, +10, 0007; reject 3.14, 1e3, 1_000, 0x10,
  10ms; no surrounding-whitespace trimming.
- number: accept 0, 3, 3.14, -3.14, +3.14, .5, -.5; reject 1e3, 1_000, NaN,
  Infinity.
- boolean: accept true/false case-insensitive only; reject 1, 0, yes, no, on,
  off, enabled, disabled.
Tests: table-driven for every accepted/rejected form in the spec.
Exit: deterministic, independently tested normalization.

### P4 — Validation engine (internal/validate)
- validate(contract, env) -> diagnostics, independent of CLI/output/source.
- Sequence: contract validity -> required presence (ENV_MISSING) -> empty
  (ENV_EMPTY) -> value validation (type, enum, const, pattern, min/maxLength,
  numeric bounds, multipleOf, format email/uri) -> unknown variables (ENV_UNKNOWN
  warnings).
- format: email and uri are syntactic only; no network/DNS/service checks.
- Secret-aware: diagnostics never embed values.
Tests: every diagnostic code covered; secret-valued env lines never surface.
Exit: complete validator with no output/source dependencies.

### P5 gap close — Diagnostics tests
- internal/diagnostic tests: shape (severity/code/variable/message), stable
  codes, Errorf/Warningf severities, deterministic ordering.
Exit: P5 covered.

### P6/P7 — Output (internal/output)
- Human renderer matching PRD example layout (variables grouped, error/warning
  separation, summary counts); deterministic; no secret values.
- JSON renderer: {"valid": bool, "diagnostics": [...]} with deterministic
  ordering, valid JSON, no ANSI sequences, no secrets.
- Golden tests: success, one error, many errors, warning-only, secret failure,
  malformed source, malformed contract.
Exit: output suitable for terminals and CI scripts.

### P8 — CLI integration (cmd/envdoctor)
- Commands: check (--env-file <path>, --environment, --json), generate example,
  init; version/help retained.
- Contract discovery (FR-001): envdoctor.schema.json in working directory, plus
  explicit --contract path.
- One source per invocation; explicit selection only; no merging.
- Exit codes: 0 valid, 1 validation failure, 2 usage/contract/source error,
  3 unexpected internal error.
- Normal check never prompts interactively.
Tests: end-to-end command tests incl. exit codes and JSON output.
Exit: contract + environment -> envdoctor check -> useful result end to end.

### P9 — Example generator (internal/generate)
- .env.example derived only from contract: default -> value; secret -> blank;
  enum/pattern/type without default -> blank; actual env values never copied;
  descriptions may become comments; deterministic ordering.
- Values round-trip through the EnvDoctor dotenv grammar (string escaping,
  integer/number/boolean formatting); inexpressible variable names fail.
- Existing files never silently overwritten (explicit error, atomic O_EXCL).
Tests: all value rules, secret non-disclosure, no-overwrite, determinism.
**Status: implemented — `go test ./internal/generate ./cmd/envdoctor` passes.**

### P10 — init
- Advisory discovery of variable names from .env / .env.example.
- Must not copy actual values; must not silently infer authoritative types,
  secret classification, or defaults. Suggestions marked advisory/reviewable.
Tests: safe discovery; no values copied; no name-based secret inference as fact.
**Status: implemented — `go test ./internal/init ./cmd/envdoctor` passes.**

### P11 — Security hardening
- Use realistic secret values (super-secret-password, postgres://user:password@host/db,
  token-like strings, newlines, quotes) and force failures; assert the values
  are absent from human output, JSON, errors, generated files, logs, snapshots.
- Audit: fmt/log/error/panic/JSON/generated-file paths for value embedding.
- Verify: no network access, no telemetry, no shell execution, no command
  substitution, no environment-dump modes.
Exit: security tests pass + manual review complete. RELEASE BLOCKER.
**Status: implemented — `go test ./...` passes, including the
cmd/envdoctor/security_test.go secret suite and the internal/security static
guards. Manual audit: no secret enumeration, no value-bearing errors, no
network/telemetry/shell/env-dump code found in shipping code (belt-and-suspenders
now enforced by internal/security/guard_test.go).**

### P12 — Cross-platform hardening
- Targets: linux amd64/arm64, windows amd64/arm64, darwin amd64/arm64.
- Verify paths, line endings, environment access, terminal/ANSI handling, UTF-8,
  generated-file formatting; env var names stay case-sensitive everywhere.
- CI matrix extended to cover build/test on all targets where runners allow.
Exit: deterministic behavior on all targets.
**Status: implemented — `go test ./...` passes; all six target
GOOS/GOARCH combos cross-compile (validated locally and in the CI crossbuild
job). Cross-platform posture is deliberate and locked by tests: LF-only output
and generated files (no raw CR anywhere), CRLF-tolerant deterministic dotenv
parsing, no ANSI output and therefore no terminal detection, UTF-8 preserved,
env-var names case-sensitive everywhere. GitHub-hosted runners expose amd64
(Linux/Windows) and arm64 (macOS) natively; remaining targets are covered by
cross-compilation with CGO disabled.

### P13 — Documentation
- Create docs/quickstart.md, docs/contract.md, docs/dotenv.md,
  docs/diagnostics.md, docs/security.md, docs/ci.md, docs/architecture.md.
- Complete README (what/why/install/example/contract/CI/security/links).
Exit: docs match implemented behavior.
**Status: implemented — all seven docs/ sub-documents exist and were written
against the actual implementation (CLI flags, human/JSON output samples, exit
codes, diagnostic table, dotenv grammar, contract profile, CI jobs, package
architecture, security invariants). README expanded to full what/why/install/
example/contract/CI/security/documentation coverage. No behavioral code
changes were required for this phase.

### P14 — Integration fixtures (testdata/)
- contracts/, dotenv/, environments/ directories.
- Scenarios: FastAPI-style, NestJS-style, Node API, frontend
  (NEXT_PUBLIC_*), secret-heavy app, CI process env, unknown variables,
  invalid values exercising every validation diagnostic.
Exit: realistic integration tests pass on the matrix.
**Status: implemented — testdata/{contracts,dotenv,environments} populate the
realistic scenarios; internal/integration consumes them through the real
pipeline (contract.Parse -> source LoadDotenv/FromProcess -> validate.Validate
-> output). `go test ./...` passes, including integration on the CI matrix. See
the repo-state bullet above for the coverage summary.

### P15 — Self-validation
- Add envdoctor.schema.json to repo; run envdoctor check on its own repo where
  appropriate. Do not invent config solely for dogfooding.
**Status: implemented — envdoctor.schema.json at the repository root contracts
the three variables the repo's own CI sets as environment configuration
(GOOS/GOARCH/CGO_ENABLED from the crossbuild job), all optional so that plain
`envdoctor check` (default contract discovery, process environment) exits 0 on
any machine while misspelled or out-of-range build settings produce
`ENV_INVALID_ENUM`/`ENV_TYPE_MISMATCH` with exit 1. `envdoctor check` was run
against the repo contract directly (human and JSON output) for every
GOOS/GOARCH crossbuild combo. No configuration was invented purely for
dogfooding. internal/integration/self_validation_test.go guards the contract
stays valid and behavior locks on the CI matrix. Dogfooding also surfaced the
UTF-8 BOM rejection, documented in docs/dotenv.md.

### P16 — Release engineering
- Semantic versioning; automated test, lint/vet, build, cross-compile matrix,
  packaging, checksums, changelog, release. Keep contract semantics, diagnostic
  codes, JSON, and exit codes as compatibility surfaces.
**Status: implemented — `version` is ldflags-injectable with a semver guard;
tools/release packages all six targets deterministically with README/LICENSE
and writes SHA256SUMS + NOTES.md (changelog section); .github/workflows/
release.yml runs the QA gate on tag push, verifies tag == embedded version,
packages, and opens a draft GitHub release; docs/release.md pins versioning
policy and the four compatibility surfaces. Verified locally: packaged native
binary reports the injected version when executed; archives are byte-identical
for identical inputs; SHA256SUMS digests match; six-target matrix covered by
tests. `go test ./...` passes.

### P17 — MVP review
- Run acceptance checklists: roadmap §20, PRD §21. Fix any failures.

## Execution Order

P1-tests -> P2 -> P3 -> P4 -> P5-tests -> P6 -> P7 -> P8 -> P9 -> P10 ->
P11 -> P12 -> P13 -> P14 -> P15 -> P16 -> P17.

First useful milestone: P8 (`envdoctor check` works end to end).

## Verification Commands

go build ./... ; go test ./... ; go vet ./... ; gofmt -l .
CI runs the same on Linux, Windows, macOS.

## Risks / Notes

- P1 contract has full table-driven coverage (parse, normalize, semantics).
- P2 dotenv duplicate-key and interpolation behavior must be explicit, failing
  loudly rather than silently resolving.
- Keep secret redaction tested, not assumed; treat env values as toxic data.
- No new dependencies unless they materially improve correctness.