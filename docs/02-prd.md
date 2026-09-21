# EnvDoctor --- Product Requirements Document

**Status:** Approved development baseline\
**Version:** 1.0\
**Date:** 2026-09-17\
**Implementation language:** Go

## 1. Product Overview

EnvDoctor is an open-source CLI that gives application repositories an
explicit configuration contract and validates actual environments
against it.

The product exists because environment configuration is usually implicit
and fragmented across source code, `.env`, `.env.example`, CI, Docker,
deployment platforms and tribal knowledge.

EnvDoctor creates a canonical contract and provides deterministic
diagnostics.

## 2. Product Vision

> **Make application configuration explicit, testable, portable, and
> safe.**

Long term, EnvDoctor should become a trusted local and CI diagnostic
layer without becoming a hosted configuration platform.

## 3. Target Users

### Primary

-   backend developers;
-   full-stack developers;
-   DevOps-oriented developers;
-   open-source maintainers.

### Secondary

Teams with:

-   CI pipelines;
-   Dockerized applications;
-   multiple services;
-   staging/production environments;
-   onboarding/configuration drift problems.

### Tertiary

Open-source repositories where contributors need immediate configuration
diagnostics.

## 4. User Problems

1.  Missing variables are discovered only at runtime.
2.  Values have wrong types or formats.
3.  `.env.example` becomes stale.
4.  Local and CI environments drift.
5.  Configuration requirements are undocumented.
6.  Diagnostic tools can expose secrets.
7.  Different languages/frameworks solve the same problem differently.

## 5. Product Goals

### G1 --- Explicit contract

Developers can define configuration requirements in
`envdoctor.schema.json`.

### G2 --- Local validation

Developers can validate a dotenv file.

### G3 --- Process validation

Developers can validate the current process environment.

### G4 --- Useful diagnostics

Failures identify variable, severity, stable code and actionable
message.

### G5 --- CI integration

The CLI provides deterministic exit codes and JSON.

### G6 --- Safe generation

Developers can generate `.env.example` without copying secrets.

### G7 --- Standalone distribution

Users can install a binary without installing Go.

### G8 --- Open-source quality

The repository includes tests, documentation, security guidance and
release automation.

## 6. Non-Goals

MVP will not:

-   manage secrets;
-   encrypt or synchronize secrets;
-   launch applications;
-   inject runtime variables;
-   connect to infrastructure;
-   inspect Docker;
-   inspect CI workflows;
-   scan source code;
-   understand frameworks;
-   provide AI assistance;
-   provide a hosted dashboard;
-   require an account;
-   send telemetry;
-   manage deployments.

## 7. Core User Journeys

### Journey A --- New repository

``` text
Install
  ↓
envdoctor init
  ↓
Review contract
  ↓
envdoctor check --env-file .env
  ↓
Fix configuration
  ↓
Pass
```

### Journey B --- CI

``` text
CI environment
  ↓
envdoctor check --environment --json
  ↓
exit 0 / non-zero
```

### Journey C --- Example generation

``` text
Contract
  ↓
envdoctor generate example
  ↓
.env.example
```

## 8. Functional Requirements

### FR-001 Contract discovery

Locate `envdoctor.schema.json` according to documented rules and support
an explicit path.

### FR-002 Contract loading

Parse JSON and validate the EnvDoctor Contract Profile.

### FR-003 Contract versioning

Reject unsupported versions clearly.

### FR-004 Required variables

Report missing required variables as `ENV_MISSING`.

### FR-005 Empty variables

Distinguish missing from present-but-empty.

### FR-006 Types

Validate string, integer, number and boolean.

### FR-007 Enum

Enforce exact enum values.

### FR-008 Pattern

Support regex constraints.

### FR-009 Numeric constraints

Support minimum, maximum, exclusive bounds and multipleOf.

### FR-010 Formats

Support email and URI.

### FR-011 Secrets

Understand `x-envdoctor-secret`.

### FR-012 Secret protection

Never emit secret values in human or JSON diagnostics.

### FR-013 Unknown variables

Warn by default.

### FR-014 Dotenv source

Support explicit dotenv files.

### FR-015 Process environment

Support the current process environment.

### FR-016 Source isolation

Use one source per MVP invocation.

### FR-017 Human output

Default output is terminal-friendly.

### FR-018 JSON output

Provide deterministic machine-readable output.

### FR-019 Exit codes

Implement stable exit codes.

### FR-020 Example generation

Generate `.env.example` from the contract.

### FR-021 Safe generation

Never generate secret values.

### FR-022 No silent overwrite

Never overwrite an existing generated file without explicit intent.

### FR-023 Determinism

Equivalent inputs produce equivalent results across OSes.

### FR-024 Offline operation

Normal validation makes no network request.

## 9. `init` Requirements

`envdoctor init` may inspect `.env` and `.env.example`.

It must:

-   avoid copying actual values;
-   avoid treating uncertain type guesses as facts;
-   avoid silently classifying secrets from names alone.

Inference may be offered as a suggestion for review.

## 10. CLI Requirements

Initial commands:

``` bash
envdoctor init
envdoctor check
envdoctor generate example
```

Examples:

``` bash
envdoctor check --env-file .env
envdoctor check --environment
envdoctor check --json
```

Normal `check` must not prompt interactively.

## 11. Output Requirements

Human output should be concise and diagnostic:

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

JSON output should be structurally stable:

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

## 12. Contract Requirements

Canonical file:

``` text
envdoctor.schema.json
```

It must:

-   declare JSON Schema 2020-12;
-   declare EnvDoctor contract version;
-   use root object type;
-   describe variables under properties;
-   use root required for requiredness.

Unsupported JSON Schema constructs are contract errors.

## 13. Security Requirements

These are release blockers.

-   Secret values must never be logged.
-   Secret values must never be in JSON diagnostics.
-   Secret values must never be generated into example files.
-   Secret values must never appear in panic/error messages.
-   There must be no generic environment-dump debug flag.
-   Normal validation must make no network request.
-   Tests must verify non-disclosure.

## 14. Reliability Requirements

The CLI must:

-   reject malformed contracts clearly;
-   reject malformed environment files clearly;
-   distinguish user errors from internal errors;
-   reject unsupported semantics instead of silently ignoring them;
-   behave consistently across Linux, Windows and macOS.

## 15. Performance Requirements

For ordinary repositories, validation should complete well under one
second.

The MVP should avoid:

-   network calls;
-   repository-wide scanning;
-   unnecessary subprocesses.

Performance should be measured before optimization.

## 16. Distribution Requirements

Initial release targets:

``` text
Linux amd64
Linux arm64
Windows amd64
Windows arm64
macOS amd64
macOS arm64
```

End users should not need Go installed.

## 17. CI Requirements

The EnvDoctor repository must use CI for:

-   unit tests;
-   integration tests;
-   formatting;
-   static analysis;
-   cross-platform build verification.

## 18. Quality Requirements

Prefer:

-   explicit interfaces;
-   small packages;
-   table-driven tests;
-   deterministic output;
-   clear errors;
-   standard library solutions;
-   minimal dependencies.

Avoid premature abstraction.

## 19. Future Direction

Possible post-MVP capabilities:

### Phase 2

-   environment diff;
-   multiple sources;
-   richer comparison;
-   stricter unknown-variable policy;
-   YAML serialization;
-   workspace support.

### Phase 3

-   repository audit;
-   Docker/Compose inspection;
-   CI analysis;
-   source-code variable discovery;
-   framework-specific diagnostics.

Later:

-   integrations;
-   IDE tooling;
-   secret-store adapters.

All future work must preserve the small safe core.

## 20. Success Metrics

Useful signals include:

-   installation success;
-   time to first successful validation;
-   repository adoption;
-   CI adoption;
-   GitHub engagement;
-   issue patterns around confusing semantics;
-   zero security incidents.

MVP should not use telemetry.

## 21. MVP Acceptance Criteria

-   [ ] Standalone Go CLI builds.
-   [ ] Contract loading/validation works.
-   [ ] Dotenv parsing works.
-   [ ] Process environment works.
-   [ ] Required/optional semantics work.
-   [ ] Empty values are distinguished.
-   [ ] All four primitive types work.
-   [ ] Enum works.
-   [ ] Pattern works.
-   [ ] Numeric constraints work.
-   [ ] URI/email formats work.
-   [ ] Unknown-variable warnings work.
-   [ ] Secret values never leak.
-   [ ] Human output works.
-   [ ] JSON output works.
-   [ ] Exit codes work.
-   [ ] Example generation works.
-   [ ] No silent overwrite.
-   [ ] `init` works safely.
-   [ ] Unit and integration tests pass.
-   [ ] Cross-platform builds pass.
-   [ ] Documentation is complete.
-   [ ] Security policy exists.
-   [ ] Release artifacts are reproducible.

## 22. Release Definition

The first public release must be explainable in one sentence:

> **Define your environment contract once, then validate any environment
> against it.**

If a feature cannot support that promise, it does not belong in MVP.
