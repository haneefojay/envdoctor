# EnvDoctor — Claude Code Working Instructions

Project Context

EnvDoctor is an open-source Go CLI for defining, validating, and diagnosing application configuration requirements.

Its core idea is:

Configuration should have a contract.

EnvDoctor defines a contract for environment variables, validates a concrete environment against that contract, and explains failures without exposing secrets.

The project is intentionally small. Protect that property.

1. Mandatory Reading Order

Before implementing any task:

Read docs/01-consolidated-specification.md.
Read docs/02-prd.md.
Read docs/03-development-roadmap.md.
Read AGENTS.md.
Inspect the current repository state and existing tests.

Use the roadmap to determine the assigned phase and scope.

Do not implement from a prompt alone when the repository documentation already defines the behavior.

2. Current Decisions Are Locked

Treat these decisions as settled unless the governing documentation is deliberately changed:

Language: Go
Canonical contract file: envdoctor.schema.json
Contract basis: JSON Schema Draft 2020-12
EnvDoctor Contract Profile: restricted subset
Primitive types: string, integer, number, boolean
Secret marker: x-envdoctor-secret
MVP sources: explicit dotenv file and process environment
Source model: one source per invocation; no source merging
Commands: init, check, generate example
Exit codes: 0 valid, 1 validation failure, 2 usage/contract/source error, 3 unexpected internal error
Unknown environment variables: warnings by default
Defaults: descriptive metadata only
Secrets: never disclosed
No service/network checks

Do not reopen these decisions during ordinary implementation work.

3. Product Boundary

EnvDoctor is a configuration contract and validation tool.

It is not:

a secrets manager;
a dotenv manager;
a deployment system;
a cloud configuration platform;
a runtime injector;
a service-health checker;
an AI assistant;
a framework configuration library.

Keep the implementation inside this boundary.

4. Engineering Priorities

When tradeoffs arise, use this priority order:

Security
Correctness
Deterministic behavior
Diagnostic quality
Simple architecture
Cross-platform behavior
Performance
Convenience

Do not trade security or deterministic behavior for convenience.

5. Contract Implementation

The contract is a restricted JSON Schema 2020-12 profile.

Supported keywords/features are explicitly defined by the specification.

Unsupported JSON Schema constructs must not be silently ignored.

A malformed, contradictory, unsupported, or unknown-version contract is a contract error.

Keep the contract parser/model separate from environment-source parsing.

The core validator should conceptually remain:

validate(contract, environment) -> diagnostics

It should not know whether the environment came from dotenv or the operating system.

6. Requiredness and Defaults

Requiredness comes from the root required array.

Required variables must:

exist;
be non-empty;
satisfy constraints.

Optional variables may be absent.

Defaults do not inject values.

The application remains responsible for applying defaults.

Treat required + default as a contract error.

Do not use defaults to make a missing required variable pass validation.

7. Secret Handling

Treat environment values as toxic data.

If a variable is marked with:

"x-envdoctor-secret": true

its actual value must never be disclosed.

Review every change that touches:

fmt.Printf;
fmt.Println;
log.*;
error construction;
panic paths;
JSON serialization;
diagnostic construction;
temporary files;
generated files;
snapshots;
debug output;
test failures.

Never add a debug mode that prints all environment variables or contract-resolved values.

For secret-related diagnostics, describe the condition without the value.

8. Environment Sources

MVP supports:

an explicit dotenv file;
the process environment.

Normalize both to the same internal representation:

map[string]string

Do not add source precedence or merging.

Do not automatically load multiple dotenv files.

Do not automatically combine a dotenv file with process environment values unless the governing specification explicitly calls for that behavior in the particular CLI operation.

9. Dotenv Parser

The parser must be deterministic and must not execute shell semantics.

Supported MVP behavior includes:

KEY=value;
comments;
blank lines;
single quotes;
double quotes;
empty values;
documented escapes;
LF;
CRLF;
Unicode.

Never execute:

shell commands;
command substitutions;
arbitrary expressions.

Duplicate keys must not be silently resolved.

Unsupported interpolation must fail clearly rather than silently producing an unexpected value.

Keep dotenv parsing isolated from contract validation.

10. Type Semantics

Environment values begin as strings.

Boolean

Accept only true and false, case-insensitively.

Reject:

1;
0;
yes;
no;
on;
off;
enabled;
disabled.
Integer

Use the grammar defined by the project specification.

Examples accepted:

0;
1;
3000;
-10;
+10;
0007.

Reject:

3.14;
1e3;
1_000;
0x10;
10ms.
Number

Examples accepted:

0;
3;
3.14;
-3.14;
+3.14;
.5;
-.5.

Reject the initially defined forms:

1e3;
1_000;
NaN;
Infinity.

Strings must not be silently trimmed or normalized.

URI/email formats are syntactic validation only.

Never make a network request as part of validation.

11. Unknown Variables

Unknown variables produce warnings by default.

Do not turn them into validation errors merely because they are not in the contract.

A normal process environment contains unrelated variables.

Keep warning/error semantics aligned with the specification.

12. Diagnostics

Diagnostic codes are stable identifiers.

Initial codes:

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

Every diagnostic must at least provide:

severity;
code;
variable;
message.

Never include secret values.

Output order must be deterministic. Sort keys or diagnostics explicitly where necessary; never depend on Go map iteration order.

13. CLI Behavior

MVP commands:

envdoctor init
envdoctor check
envdoctor generate example

Human output should be concise and actionable.

JSON output must be machine-readable and deterministic.

Do not emit ANSI terminal escape sequences in JSON.

Exit codes:

0 valid;
1 validation failure;
2 usage/contract/source error;
3 unexpected internal error.

Do not casually alter exit-code semantics.

14. Determinism

The same contract and same environment should produce equivalent diagnostics and machine-readable output regardless of:

operating system;
Go map iteration order;
line-ending convention where inputs are semantically equivalent;
terminal presence.

Be explicit about ordering in:

diagnostics;
generated files;
JSON output;
discovered variables.

Avoid time-dependent or random output unless explicitly required.

15. Example Generation

The generated .env.example must come only from contract metadata.

Rules:

explicit default may be emitted;
secrets must remain blank;
variables without defaults remain blank;
enum alone does not create a fake value;
pattern alone does not create a fake value;
type alone does not create a fake value;
descriptions may become comments;
ordering is deterministic;
existing files must not be overwritten silently.

Never copy actual environment values.

Never invent secret placeholders that could be mistaken for real credentials.

16. init

init helps users adopt EnvDoctor.

It may discover variable names, but it must not silently decide:

actual values;
authoritative types;
secret classification;
defaults.

Any inference should be clearly advisory and subject to user review.

Do not make init an uncontrolled reverse-engineering engine.

17. Architecture Guidance

Prefer a small package structure such as:

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

Use only the packages actually justified by the implementation.

Keep:

contract parsing;
source parsing;
normalization;
validation;
diagnostic construction;
output formatting

as separate responsibilities.

Do not create interfaces for hypothetical future implementations unless the boundary is already useful.

18. Dependencies

Prefer the Go standard library.

A dependency should be added only when it materially improves correctness or removes significant complexity.

Before adding one, consider:

whether the standard library already solves the problem;
maintenance burden;
binary size;
security implications;
cross-platform behavior;
whether the dependency introduces unnecessary behavior.

Do not add dependencies merely because they are popular.

19. Testing Strategy

Use table-driven tests where they improve clarity.

Every implementation phase should add tests for its behavior.

Important coverage includes:

contract parsing and validation;
unsupported features;
required/default conflicts;
secret/default conflicts;
dotenv grammar;
duplicate dotenv keys;
type grammar;
every diagnostic code;
numeric constraints;
enum behavior;
pattern behavior;
URI/email formats;
JSON output;
exit codes;
generated examples;
init;
secret non-disclosure;
Unicode;
CRLF/LF;
Windows/macOS/Linux behavior where practical.

Tests must never accidentally print secrets.

When testing secret handling, deliberately use realistic secret-looking values and assert that those values are absent from every user-visible representation.

20. Security Review Before Completion

Before declaring a task complete, inspect the changed code for accidental disclosure through:

formatted output;
errors;
logs;
panic messages;
JSON marshaling;
generated files;
temporary files;
snapshots;
test output;
debug statements.

Also verify that the implementation does not introduce:

shell execution;
command substitution;
network requests;
telemetry;
unsafe temporary-file behavior;
nondeterministic secret exposure.

Security behavior should be tested, not merely assumed.

21. Error Handling

Errors should be:

actionable;
deterministic;
safe;
appropriately classified.

Do not wrap an underlying error in a way that leaks a secret value.

Do not use panic for ordinary user/configuration errors.

Contract/source/user errors should map to the documented exit-code behavior.

Unexpected internal failures should remain distinguishable from validation failures.

22. Implementation Workflow

For each roadmap task:

Read the relevant specification and roadmap phase.
Inspect the current code.
Identify existing abstractions before creating new ones.
Implement the smallest complete change.
Add focused tests.
Run tests.
Review security implications.
Review deterministic behavior.
Review cross-platform behavior.
Review scope.
Update documentation only where the implementation genuinely changes documented behavior.
Report the result concisely.

Do not refactor unrelated code during a feature task unless the existing structure makes the required implementation unsafe or impossible.

23. Roadmap Order

Follow the roadmap rather than inventing an alternative implementation sequence.

The planned phases are:

P0 Project bootstrap
P1 Contract model
P2 Environment sources
P3 Type normalization
P4 Validation engine
P5 Diagnostics
P6 Human output
P7 JSON output
P8 CLI integration
P9 Example generator
P10 init
P11 Security hardening
P12 Cross-platform hardening
P13 Documentation
P14 Integration fixtures
P15 Self-validation
P16 Release engineering
P17 MVP review

If a later phase appears necessary for a current phase, inspect the roadmap and specification before changing order.

24. Scope-Control Rules

Do not implement these as MVP functionality:

secret managers;
encryption;
secret sync;
remote configuration;
dashboards;
authentication;
telemetry;
AI;
runtime injection;
envdoctor run;
contract profiles;
schema imports/inheritance;
cross-variable expressions;
arbitrary custom validators;
Docker analyzers;
CI analyzers;
source-code scanning;
framework-specific analyzers;
deployment management;
network/service health checks.

Do not build speculative infrastructure for them.

The core must remain useful without any of them.

25. When to Stop and Reassess

Stop and reassess only when there is a genuine contradiction between:

the governing specification;
the PRD;
the roadmap;
existing locked decisions;
security requirements.

Do not stop merely because an ordinary implementation detail is unspecified.

For ordinary ambiguity, choose the smallest deterministic, secure behavior consistent with the existing contract.

Do not silently create new product semantics.

26. Definition of Done

Before marking work complete:

the requested roadmap scope is implemented;
the code compiles;
relevant tests pass;
meaningful edge cases are covered;
secret values cannot leak;
diagnostics are stable and safe;
output is deterministic;
exit codes match the specification;
unsupported behavior fails clearly;
cross-platform implications have been considered;
no unrelated scope has been introduced.
27. Final Instruction

Build EnvDoctor as a small, trustworthy core.

Do not optimize for the number of features implemented.

Optimize for:

correct configuration contracts, safe diagnostics, deterministic behavior, and developer trust.
