# EnvDoctor Agent Instructions

## 1. Purpose

EnvDoctor is an open-source, local-first developer CLI for defining, validating, and diagnosing application configuration requirements.

The central product principle is:

> **Configuration should have a contract.**

EnvDoctor defines the contract between an application and its environment, validates an environment against that contract, and explains configuration failures without exposing secrets.

These instructions apply to every coding agent working in this repository.

---

## 2. Read These Documents First

Before implementing or changing project behavior, read these documents in this order:

1. `docs/01-consolidated-specification.md`
2. `docs/02-prd.md`
3. `docs/03-development-roadmap.md`
4. This `AGENTS.md`
5. `CLAUDE.md` when using Claude Code

Then inspect the existing implementation and tests.

The specification, PRD, and roadmap are the governing product documents. Do not invent requirements that contradict them.

---

## 3. Current Technology Decision

The programming language is **Go (Golang)**.

This decision is final.

Prefer the Go standard library. Add third-party dependencies only when they materially improve correctness or remove significant complexity.

EnvDoctor must build as a standalone executable with no runtime dependency on another language or runtime.

---

## 4. Product Identity

EnvDoctor is:

- a configuration-contract tool;
- repository-level;
- local-first;
- offline for normal validation;
- source-independent at its core;
- read-only by default;
- security-conscious;
- deterministic;
- language-agnostic.

EnvDoctor is not:

- a secrets manager;
- a dotenv manager;
- a deployment platform;
- a cloud configuration service;
- a runtime configuration injector;
- a service-health checker;
- an AI configuration assistant;
- a framework-specific configuration system.

Do not allow implementation decisions to gradually turn EnvDoctor into one of those products.

---

## 5. Core Product Principle

The contract is the source of truth.

The environment is evaluated against the contract.

The validator should conceptually operate as:

```text
contract + environment -> diagnostics

The core validation logic must not depend on whether the environment came from a dotenv file, process environment, or a future source.

6. Contract Format

The canonical contract file is:

envdoctor.schema.json

It uses JSON Schema Draft 2020-12, but EnvDoctor supports a deliberately restricted EnvDoctor Contract Profile.

Do not silently accept arbitrary JSON Schema features.

Unsupported or invalid contract constructs must produce a contract error.

Supported environment variable primitive types:

string
integer
number
boolean

MVP does not support object, array, or null environment-variable types.

Supported contract keywords/features include:

type
required
enum
const
default
description
title
pattern
minLength
maxLength
minimum
maximum
exclusiveMinimum
exclusiveMaximum
multipleOf
format
x-envdoctor-secret

Supported formats:

email
uri

Do not silently normalize unsupported schema constructs into something else.

7. Contract Semantics

Requiredness is expressed using the root required array.

A required variable:

must exist;
must be non-empty;
must satisfy its declared constraints.

An optional variable may be absent.

Defaults are descriptive metadata only. EnvDoctor does not inject defaults into the process environment.

The application remains responsible for applying actual runtime defaults.

The following are contract errors:

a required variable has a default;
a secret variable has a default;
invalid or contradictory constraints;
duplicate variable definitions;
unsupported contract features;
unsupported/unknown contract versions.

Enum matching is exact and case-sensitive after type interpretation.

Patterns must use the deterministic regex semantics provided by the Go implementation.

8. Secret Handling

x-envdoctor-secret: true marks a variable as sensitive.

Secret handling is a security invariant, not a presentation preference.

Actual secret values must never appear in:

stdout;
stderr;
diagnostics;
JSON output;
logs;
panic messages;
errors;
generated files;
snapshots;
test failure output;
debug output.

Never add a generic "dump the environment" or "verbose secrets" mode.

Do not rely on secret-name heuristics as the primary security mechanism. Explicit contract classification is authoritative for MVP.

If a future feature could expose a secret, redesign it before implementation.

9. Environment Sources

MVP supports:

an explicitly selected dotenv file;
the process environment.

The source layer should normalize values into:

map[string]string

The validator should operate on this normalized representation.

MVP does not define source merging or precedence between multiple environment sources.

Do not introduce .env, .env.local, .env.production, process-environment, and other precedence rules unless the specification is explicitly expanded.

10. Dotenv Rules

The dotenv parser must have explicit, deterministic behavior.

MVP supports:

KEY=value;
blank lines;
comments;
single-quoted values;
double-quoted values;
empty values;
documented escapes;
LF and CRLF line endings;
Unicode/UTF-8.

The parser must not:

execute shell commands;
execute command substitutions;
evaluate arbitrary shell syntax;
silently accept unsupported dialect behavior;
silently resolve duplicate keys.

Interpolation is not a core EnvDoctor feature. Unsupported interpolation must produce a clear parser/source error rather than silently generating a different value.

Malformed dotenv input must be reported as a source/environment-file error.

11. Value Semantics

Environment variables arrive at the validation boundary as strings.

Boolean

Accept:

true
false

Matching is case-insensitive.

Do not accept as booleans:

1
0
yes
no
on
off
enabled
disabled
Integer

The grammar must be deterministic.

Examples accepted by the contract semantics:

0
1
3000
-10
+10
0007

Examples rejected:

3.14
1e3
1_000
0x10
10ms
Number

Examples accepted:

0
3
3.14
-3.14
+3.14
.5
-.5

Initially rejected:

1e3
1_000
NaN
Infinity

Strings must preserve their parsed value. Do not silently trim or normalize string values.

format: uri and format: email are syntactic validations only.

Never perform network requests, DNS checks, or service-health checks as part of validation.

12. Unknown Variables

Unknown environment variables are warnings by default.

This is intentional because normal process environments contain unrelated variables such as PATH, HOME, CI, and operating-system variables.

Unknown variables must not cause validation failure by default.

Do not invent an "all warnings are errors" policy unless the specification is explicitly changed.

13. Diagnostics

Diagnostics must be structured and stable.

Each diagnostic contains at least:

severity;
code;
variable;
message.

Diagnostic codes are stable API-level identifiers. Messages may evolve.

Initial codes include:

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

Never include actual secret values in diagnostic messages.

Output ordering must be deterministic. Never rely on Go map iteration order.

14. Exit Codes

Use these stable exit codes:

0 — environment valid;
1 — validation failure;
2 — usage, contract, or source error;
3 — unexpected internal error.

Do not change these codes casually.

An environment is valid only when:

the contract is valid;
required variables are present;
required variables are non-empty;
values satisfy their constraints;
there are no error-severity diagnostics.

Warnings do not invalidate the environment by default.

15. CLI Scope

MVP commands are:

envdoctor init
envdoctor check
envdoctor generate example

check must support the MVP environment-source model.

The CLI must provide:

useful human-readable output;
machine-readable JSON output;
stable exit codes.

JSON output must not contain ANSI terminal formatting.

Do not add unrelated commands simply because they are convenient.

16. Example Generation

.env.example is derived only from the contract.

Rules:

explicit default -> generated value may use the default;
secret -> blank;
no default -> blank;
enum without default -> blank;
pattern without default -> blank;
type without default -> blank;
actual environment values are never copied;
descriptions may become comments;
output ordering is deterministic;
existing files must not be silently overwritten.

Never generate fake secret values.

Do not infer an example value merely because an enum or pattern makes one possible.

17. init

envdoctor init is an adoption assistant, not an oracle.

It may discover variable names from sources such as .env or .env.example, but:

it must never copy actual environment values into the contract;
it must not silently infer authoritative types;
it must not silently infer secret classification;
it must not silently infer defaults.

Suggestions may be presented for user review, but suggestions must remain distinguishable from authoritative contract data.

18. Architecture Rules

Prefer a small architecture with clear boundaries.

A suitable conceptual structure is:

cmd/
  envdoctor/

internal/
  contract/
  source/
  normalize/
  validate/
  diagnostic/
  generate/
  output/

Do not create speculative packages.

The core validator should remain independent of CLI presentation and source-specific parsing.

A useful conceptual pipeline is:

Configuration Source
        ↓
Source Parser
        ↓
Normalized Environment
        ↓
Contract Validator
        ↓
Diagnostics
        ↓
Human / JSON Output

Keep parsing, validation, diagnostics, and presentation separate.

19. Testing Requirements

Tests are part of the product contract.

Test:

Contract
valid contracts;
malformed contracts;
unsupported keywords;
invalid versions;
duplicate variables;
required/default conflicts;
secret/default conflicts;
invalid constraints;
enum/type conflicts.
Dotenv
comments;
blank lines;
quoting;
empty values;
malformed lines;
duplicate keys;
unsupported interpolation;
CRLF;
LF;
Unicode.
Types

Test every accepted and rejected boolean, integer, and number representation defined by the specification.

Validation

Every diagnostic code must have coverage.

Security

Use realistic secret values and verify that they never appear in:

human output;
JSON;
generated files;
errors;
logs;
snapshots;
test output.
Cross-platform

Cover:

Linux;
Windows;
macOS;
path behavior;
line endings;
environment access;
terminal behavior;
UTF-8;
generated files.

Prefer table-driven Go tests where appropriate.

20. Cross-Platform Requirements

EnvDoctor must behave deterministically across Linux, Windows, and macOS.

Environment variable names are semantically case-sensitive regardless of host operating-system behavior.

Do not allow Windows case-insensitive environment behavior to create a different contract interpretation.

Be deliberate about:

path separators;
file encoding;
line endings;
terminal detection;
ANSI handling;
environment access;
generated-file formatting.
21. Performance

EnvDoctor is a local CLI and should feel immediate on normal repositories.

Do not sacrifice correctness or security for micro-optimizations.

Avoid unnecessary:

network access;
subprocesses;
shell execution;
repeated file reads;
large allocations.

Do not introduce concurrency unless it provides a meaningful benefit without making deterministic behavior or security harder.

22. Code Quality

Write idiomatic Go.

Prefer:

small functions;
explicit control flow;
clear types;
useful errors;
table-driven tests;
standard-library solutions;
deterministic output.

Avoid:

speculative abstractions;
premature generalization;
global mutable state;
hidden side effects;
unnecessary reflection;
clever code that obscures security behavior.

Error messages should help developers fix the problem without exposing sensitive data.

23. Scope Control

The following are outside MVP unless the governing documents are explicitly updated:

secret management;
encryption;
secret synchronization;
cloud configuration;
dashboard;
authentication;
telemetry;
AI;
runtime injection;
envdoctor run;
contract profiles;
schema inheritance/imports;
cross-variable expressions;
arbitrary custom validators;
Docker analysis;
CI configuration analysis;
source-code scanning;
framework-specific analyzers;
deployment management;
network/service health checks.

diff and audit are future directions, not reasons to complicate the MVP architecture.

Do not implement future features "because the architecture will need them later." Keep extension points small and justified.

24. Agent Workflow

For every assigned task:

Read the governing documents.
Read the relevant roadmap phase.
Inspect existing code before changing it.
Identify the smallest correct implementation.
Implement only the assigned scope.
Add or update focused tests.
Run the relevant test suite.
Run broader tests when practical.
Review changed code for secret leakage.
Review changed code for nondeterministic behavior.
Review changed code for cross-platform issues.
Check that no scope creep was introduced.
Report what changed, tests run, and any remaining issue.

Do not rewrite unrelated code merely to improve style.

Do not modify product decisions while implementing a roadmap phase.

25. Handling Ambiguity

When implementation details are not explicitly specified:

prefer the simplest deterministic behavior;
preserve the security invariants;
preserve cross-platform behavior;
avoid introducing new product semantics;
follow established contract semantics;
inspect existing tests and surrounding code;
document a genuine unresolved product decision rather than silently inventing one.

Do not reopen decisions that are already explicitly locked.

26. Definition of Done

A task is not complete merely because the code compiles.

Before declaring completion:

implementation matches the specification;
tests cover meaningful behavior;
relevant tests pass;
errors are safe;
secrets cannot leak;
output is deterministic;
exit codes are correct;
unsupported features fail clearly;
cross-platform behavior has been considered;
no unrelated scope has been added.
27. Final Rule

Build EnvDoctor as a small, trustworthy configuration-contract engine first.

Correctness, security, deterministic behavior, and clear diagnostics matter more than feature count.

When in doubt, prefer the smallest implementation that faithfully enforces the contract.
