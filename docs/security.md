# EnvDoctor Security

EnvDoctor is a configuration-contract tool that frequently stands next to
real secrets in `.env` files, contract files, and process environments. The
product promises that **secret values never appear in any output**. That is a
security invariant, not a presentation preference.

Realistic secret values never appear in: stdout; stderr; diagnostics; JSON
output; logs; panic messages; errors; generated files; snapshots; test failure
output; debug output.

## Why it is architecturally enforced

The safety boundary is **diagnostics**. The validator, the output layer, and
the source parsers never embed an environment *value* in a diagnostic message
or error - only variable names, codes, and fixed text. The output layer never
reads the environment directly; it renders only the diagnostics it receives.
There is therefore no code path through which a value could reach output.

The invariants that support this:

- **Diagnostics are value-free.** Every message is fixed text or references
  only names, codes, and counts.
- **No environment-dump mode.** There is no "print the whole environment" or
  "verbose secrets" command, now or as a future convenience.
- **Errors are value-free.** Parse and IO errors report file paths and line
  numbers, never values.
- **`generate` never emits secrets.** `envdoctor generate example` writes
  `API_KEY=` for a secret; a secret with a default violates the contract (a
  secret default is itself a contract error). No fake secret values are ever
  generated.
- **`init` copies nothing.** `envdoctor init` discovers variable *names* only;
  values are discarded by construction.
- **No secret-name heuristics.** Explicit contract classification
  (`x-envdoctor-secret`) is authoritative. EnvDoctor does not guess secrecy
  from a variable's name as its primary mechanism.
- **No network.** `format: email` and `format: uri` are syntactic checks only.
  Validation never makes network requests, DNS lookups, or service-health
  checks.

## Offline enforcement: the security guard package

`internal/security` is a test-only package that statically scans every
non-test Go file under `cmd/` and `internal/` and fails the build if shipping
code contains a banned capability:

- network and TLS: `net/http`, `net`, `crypto/tls`, `http.` client calls,
  `net.Dial`, `net.Listen`;
- subprocesses: `os/exec`, `os.StartProcess`, `syscall.Exec`, `exec.Command`;
- logging frameworks and telemetry: `log`, `log/slog`, `telemetry`, sentry,
  segment, mixpanel, amplitude, newrelic markers;
- panic: `panic(` in ordinary error paths;
- environment dumping: `os.Environ()` anywhere other than the documented
  process source (`internal/source/process.go`).

These guards are belt-and-suspenders under the behavior-level secret tests.

## Behavior-level secret tests

`cmd/envdoctor/security_test.go` runs the real CLI with realistic secret values
(`super-secret-password`, a PostgreSQL-style connection string, token-like
strings, values with newlines and quotes) and forces every value-driven
diagnostic in both human and JSON modes, plus malformed-file, `init`, and
`generate` paths. In each case it asserts the secret value never reaches
stdout, stderr, or a generated file. The `generate` and `init` packages have
their own non-disclosure tests as well.

## When changing code

- Treat environment values as toxic data everywhere.
- Never add a value to a diagnostic message, error, or log line - even
  transiently.
- Do not add generic dump/verbose modes.
- If a future feature could expose a secret, redesign it before implementing.
- Run `go test ./...`; the security suites and guards run as part of the
  normal test suite in CI on every supported platform.