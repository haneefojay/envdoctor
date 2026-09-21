# EnvDoctor --- Development Roadmap

**Status:** Implementation baseline\
**Version:** 1.0\
**Language:** Go

## 1. Development Philosophy

Implementation follows:

``` text
Specification
   ↓
PRD
   ↓
Roadmap phase
   ↓
Existing implementation
   ↓
Implement assigned phase
   ↓
Test
   ↓
Security/scope review
   ↓
Handoff
```

Agents must not:

-   invent requirements;
-   expand scope;
-   add framework-specific features;
-   add secrets management;
-   add network behavior;
-   silently change contract semantics;
-   support unsupported JSON Schema features;
-   print environment values for debugging.

A genuine contradiction must be documented and resolved at the smallest
affected boundary.

------------------------------------------------------------------------

# 2. Repository Structure

Recommended initial repository:

``` text
envdoctor/
├── cmd/
│   └── envdoctor/
│       └── main.go
├── internal/
│   ├── contract/
│   ├── source/
│   ├── normalize/
│   ├── validate/
│   ├── diagnostic/
│   ├── generate/
│   └── output/
├── testdata/
│   ├── contracts/
│   ├── dotenv/
│   └── environments/
├── docs/
├── .github/
│   └── workflows/
├── go.mod
├── go.sum
├── README.md
├── LICENSE
├── SECURITY.md
├── CONTRIBUTING.md
└── CHANGELOG.md
```

Do not create speculative packages.

------------------------------------------------------------------------

# 3. Phase 0 --- Project Bootstrap

## Goal

Create a clean buildable Go repository.

## Tasks

-   initialize Go module;
-   create CLI entry point;
-   establish supported Go version;
-   add formatting/static analysis;
-   add test infrastructure;
-   add CI;
-   add license;
-   create documentation skeleton;
-   establish repository conventions.

## Exit criteria

``` bash
go build ./...
go test ./...
```

pass locally and in CI.

------------------------------------------------------------------------

# 4. Phase 1 --- Contract Model

## Goal

Implement the EnvDoctor Contract Profile.

## Tasks

1.  Define normalized internal contract structures.
2.  Load JSON.
3.  Validate root structure.
4.  Validate EnvDoctor contract version.
5.  Validate supported schema keywords.
6.  Reject unsupported keywords.
7.  Validate required declarations.
8.  Validate types.
9.  Validate defaults.
10. Validate secret semantics.
11. Validate enum/type compatibility.
12. Validate numeric constraints.
13. Validate string constraints.
14. Validate formats.

## Critical design rule

Do not pass arbitrary JSON Schema structures throughout the application.

Normalize:

``` text
JSON contract
    ↓
EnvDoctor Contract
```

## Tests

Use table-driven tests for every accepted and rejected construct.

## Exit criteria

Valid contracts load into a normalized internal representation and
invalid contracts fail deterministically.

------------------------------------------------------------------------

# 5. Phase 2 --- Environment Sources

## Goal

Implement source abstraction.

Conceptually:

``` text
EnvironmentSource
       ↓
Load()
       ↓
NormalizedEnvironment
```

## Source A --- Process environment

Requirements:

-   preserve names;
-   preserve values;
-   do not case-normalize;
-   do not trim.

## Source B --- Dotenv

Implement the documented EnvDoctor dotenv subset.

Test:

-   comments;
-   blank lines;
-   simple values;
-   quoting;
-   empty values;
-   escapes;
-   malformed lines;
-   duplicate keys;
-   unsupported interpolation;
-   CRLF/LF;
-   Unicode.

## Exit criteria

Both sources produce the same normalized representation and the
validator remains source-agnostic.

------------------------------------------------------------------------

# 6. Phase 3 --- Type Normalization

## Goal

Convert source strings into semantic values.

Implement:

``` text
string
integer
number
boolean
```

## Integer parser

Explicitly test:

-   zero;
-   positive;
-   negative;
-   plus sign;
-   leading zeros;
-   decimals;
-   scientific notation;
-   hexadecimal;
-   whitespace;
-   malformed strings.

## Number parser

Test all approved/rejected forms.

## Boolean parser

Accept only case-insensitive `true`/`false`.

Reject all other boolean-like strings.

## Exit criteria

Normalization is deterministic and independently tested.

------------------------------------------------------------------------

# 7. Phase 4 --- Validation Engine

## Goal

Implement:

``` text
validate(contract, environment) → diagnostics
```

## Validation sequence

Recommended:

1.  contract validity;
2.  required/missing;
3.  empty;
4.  declared value validation;
5.  unknown-variable warnings.

## Rules

Implement:

-   required;
-   empty;
-   type;
-   enum;
-   const;
-   pattern;
-   min/max length;
-   numeric bounds;
-   multipleOf;
-   URI;
-   email;
-   secret-aware diagnostics.

## Exit criteria

The complete validator works without CLI/output dependencies.

------------------------------------------------------------------------

# 8. Phase 5 --- Diagnostics

## Goal

Create stable structured diagnostics.

Minimum:

``` text
severity
code
variable
message
```

Potential internal fields:

``` text
source
path
expected
```

but never actual secrets.

Implement the approved codes:

``` text
CONTRACT_INVALID
CONTRACT_UNSUPPORTED_KEYWORD
CONTRACT_DUPLICATE_VARIABLE
ENV_MISSING
ENV_EMPTY
ENV_UNKNOWN
ENV_TYPE_MISMATCH
ENV_INVALID_ENUM
ENV_PATTERN_MISMATCH
ENV_NUMBER_OUT_OF_RANGE
ENV_INVALID_FORMAT
ENV_FILE_INVALID
SOURCE_INVALID
```

## Exit criteria

Diagnostics are structured and presentation-independent.

------------------------------------------------------------------------

# 9. Phase 6 --- Human Output

## Goal

Make failures immediately understandable.

Example:

``` text
Environment validation failed

✗ DATABASE_URL
  ERROR ENV_MISSING
  Required variable is not set.

✗ PORT
  ERROR ENV_TYPE_MISMATCH
  Expected integer.

⚠ EXTRA_SETTING
  WARN ENV_UNKNOWN
  Variable is not declared in the contract.

2 errors, 1 warning
```

## Requirements

-   concise;
-   deterministic;
-   readable;
-   no secret values;
-   useful grouping.

## Tests

Golden tests for:

-   success;
-   one error;
-   many errors;
-   warning only;
-   secret failure;
-   malformed source;
-   malformed contract.

------------------------------------------------------------------------

# 10. Phase 7 --- JSON Output

## Goal

Create CI-friendly output.

Example:

``` json
{
  "valid": false,
  "diagnostics": [
    {
      "severity": "error",
      "code": "ENV_MISSING",
      "variable": "DATABASE_URL",
      "message": "Required variable is not set."
    }
  ]
}
```

## Requirements

-   valid JSON;
-   deterministic ordering;
-   no ANSI terminal sequences;
-   no secret values;
-   stable semantics.

## Exit criteria

Output is suitable for shell scripts and CI.

------------------------------------------------------------------------

# 11. Phase 8 --- CLI Integration

## Goal

Connect core engine to the CLI.

Implement:

``` bash
envdoctor check
envdoctor check --env-file .env
envdoctor check --environment
envdoctor check --json
```

## Exit codes

``` text
0 success
1 validation failure
2 usage/contract/source failure
3 unexpected internal failure
```

## Exit criteria

Complete end-to-end validation works.

------------------------------------------------------------------------

# 12. Phase 9 --- Example Generator

## Goal

Generate a safe `.env.example`.

Rules:

-   default → generate default;
-   secret → blank;
-   enum without default → blank;
-   pattern without default → blank;
-   type without default → blank;
-   description → comment;
-   actual environment values → never copied.

Output ordering must be deterministic.

Existing files must not be silently overwritten.

## Exit criteria

Generation is deterministic, safe and tested.

------------------------------------------------------------------------

# 13. Phase 10 --- `init`

## Goal

Reduce adoption friction.

Potential discovery inputs:

-   `.env`;
-   `.env.example`.

## Safety rules

Never copy:

``` text
API_KEY=actual-secret
```

into the contract.

Do not silently infer that a variable is secret from names such as
`SECRET`, `TOKEN`, `PASSWORD`, or `KEY`.

If inference exists, it is advisory and reviewable.

## Exit criteria

A typical repository can reach a usable contract quickly and safely.

------------------------------------------------------------------------

# 14. Phase 11 --- Security Hardening

Mandatory before public release.

## Tasks

Audit:

-   stdout;
-   stderr;
-   logging;
-   errors;
-   panic paths;
-   JSON serialization;
-   generated files;
-   test snapshots.

## Security tests

Use realistic secret values, including:

``` text
super-secret-password
postgres://user:password@host/db
token-like values
newlines
quotes
```

Force failures and verify the values never appear in output.

Also verify:

-   no network access;
-   no telemetry;
-   no shell execution;
-   no command substitution;
-   no environment dumps.

## Exit criteria

All security tests pass and a manual security review is complete.

------------------------------------------------------------------------

# 15. Phase 12 --- Cross-Platform Hardening

Target:

``` text
Linux amd64
Linux arm64
Windows amd64
Windows arm64
macOS amd64
macOS arm64
```

Test:

-   paths;
-   line endings;
-   environment access;
-   terminal output;
-   UTF-8;
-   generated files.

------------------------------------------------------------------------

# 16. Phase 13 --- Documentation

Required:

``` text
README.md
docs/quickstart.md
docs/contract.md
docs/dotenv.md
docs/diagnostics.md
docs/security.md
docs/ci.md
docs/architecture.md
CONTRIBUTING.md
SECURITY.md
CHANGELOG.md
```

README must immediately explain:

1.  what EnvDoctor is;
2.  why it exists;
3.  installation;
4.  quick example;
5.  contract example;
6.  CI usage;
7.  security guarantee;
8.  documentation links.

------------------------------------------------------------------------

# 17. Phase 14 --- Integration Fixtures

Create realistic fixtures.

## FastAPI-style backend

``` text
APP_ENV
PORT
DATABASE_URL
JWT_SECRET
DEBUG
```

## NestJS-style backend

``` text
NODE_ENV
PORT
DATABASE_URL
JWT_SECRET
```

## Node API

``` text
NODE_ENV
PORT
DATABASE_URL
```

## Frontend

``` text
NEXT_PUBLIC_API_URL
NEXT_PUBLIC_APP_NAME
```

## Secret-heavy application

Explicitly test non-disclosure.

## CI environment

Validate process environment.

## Unknown variables

Verify warnings.

## Invalid values

Exercise every validation diagnostic.

------------------------------------------------------------------------

# 18. Phase 15 --- Self-Validation

Once the tool has a stable contract, use EnvDoctor to validate its own
repository configuration where appropriate.

Do not invent configuration solely for dogfooding.

------------------------------------------------------------------------

# 19. Phase 16 --- Release Engineering

Automate:

1.  test;
2.  lint/static analysis;
3.  build;
4.  cross-compile;
5.  package;
6.  checksum;
7.  release;
8.  changelog.

Use semantic versioning.

Contract semantics, diagnostic codes and machine-readable output are
compatibility-sensitive.

------------------------------------------------------------------------

# 20. Phase 17 --- MVP Review

## Product

-   [ ] Solves configuration-contract validation.
-   [ ] Remains language agnostic.
-   [ ] Core is small.

## Contract

-   [ ] Contract is authoritative.
-   [ ] Unsupported semantics are rejected.
-   [ ] Defaults are non-injecting.
-   [ ] Secrets are explicit.

## Security

-   [ ] No command prints secrets.
-   [ ] JSON cannot contain secrets.
-   [ ] Generated files cannot contain secrets.
-   [ ] Normal operation makes no network requests.

## UX

-   [ ] Failures are immediately understandable.
-   [ ] Codes are stable.
-   [ ] Warnings are distinct from errors.

## CI

-   [ ] Exit codes are deterministic.
-   [ ] JSON is machine-readable.

## Distribution

-   [ ] Standalone binaries exist.
-   [ ] Users do not need Go installed.

------------------------------------------------------------------------

# 21. Agent Execution Rules

Every implementation agent should:

1.  read the specification;
2.  read the PRD;
3.  read this roadmap;
4.  inspect the existing implementation;
5.  implement only the assigned phase;
6.  run relevant tests;
7.  review security/scope;
8.  provide a concise handoff.

Do not jump ahead.

Examples:

-   Phase 2 must not implement Docker.
-   Phase 4 must not implement cross-variable conditions.
-   Phase 10 must not become an AI inference system.
-   Phase 11 must not add secret management.

------------------------------------------------------------------------

# 22. Definition of Done

A phase is complete only when:

1.  implementation exists;
2.  tests exist;
3.  tests pass;
4.  documentation is updated where necessary;
5.  behavior matches the specification;
6.  no scope creep exists;
7.  security implications are reviewed;
8.  the next phase can build on it.

------------------------------------------------------------------------

# 23. Recommended Sequence

``` text
P0  Bootstrap
 ↓
P1  Contract model
 ↓
P2  Environment sources
 ↓
P3  Normalization
 ↓
P4  Validation engine
 ↓
P5  Diagnostics
 ↓
P6  Human output
 ↓
P7  JSON output
 ↓
P8  CLI integration
 ↓
P9  Example generation
 ↓
P10 Init
 ↓
P11 Security hardening
 ↓
P12 Cross-platform
 ↓
P13 Documentation
 ↓
P14 Integration tests
 ↓
P15 Self-validation
 ↓
P16 Release engineering
 ↓
P17 MVP review
```

The first genuinely useful workflow should exist around P8:

``` text
contract + environment → envdoctor check → useful result
```

Do not wait for future features before making this usable.

------------------------------------------------------------------------

# 24. Post-MVP Backlog

## Environment diff

Potential:

``` bash
envdoctor diff
```

Must compare structurally and never expose secrets.

## Multiple sources

Only after explicit precedence semantics are designed.

## Strict unknown-variable policy

Introduce a deliberate warning/error policy rather than simply
converting all warnings to errors.

## Repository audit

Potential future:

``` bash
envdoctor audit
```

May inspect:

-   `.env`;
-   `.env.example`;
-   Docker;
-   CI;
-   source code.

This remains a layer above the core validator.

## Future integrations

Potential:

-   Docker;
-   GitHub Actions;
-   Kubernetes;
-   frameworks;
-   IDEs.

All integrations must consume the same core contract engine.

------------------------------------------------------------------------

# 25. Scope Guard

Any proposal introducing one of these requires architectural review:

``` text
remote server
database
authentication
dashboard
secret manager
encryption
AI
network requests
framework-specific parsing
runtime injection
cross-variable expression engine
schema inheritance
schema imports
automatic secret discovery
automatic source-code scanning
```

Default MVP answer:

> **Not yet.**

------------------------------------------------------------------------

# 26. Final Development Principle

EnvDoctor should become useful by being **small and trustworthy**, not
by being large.

The first release should do one thing exceptionally well:

> **Given a configuration contract and an environment, determine whether
> the environment satisfies the contract and explain exactly why when it
> does not---without leaking secrets.**
