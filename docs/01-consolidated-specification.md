# EnvDoctor --- Consolidated Specification

**Status:** Development baseline\
**Version:** 1.0 planning baseline\
**Date:** 2026-09-17\
**Implementation language:** Go\
**Product type:** Open-source, local-first developer CLI

## 1. Executive Summary

EnvDoctor is an open-source CLI for defining, validating, and diagnosing
the configuration contract between an application and its environment.

The central problem is not simply `.env` file quality. The deeper
problem is that an application's configuration requirements are
frequently implicit and fragmented across source code, `.env`,
`.env.example`, CI, Docker configuration, deployment configuration,
README files, and individual developer knowledge.

EnvDoctor establishes a repository-level configuration contract and
answers:

> **Does this environment satisfy the application's declared
> configuration contract?**

The product is intentionally:

-   language agnostic;
-   local-first;
-   offline for normal validation;
-   read-only by default;
-   safe around secrets;
-   deterministic across operating systems;
-   useful in local development and CI;
-   small at its core;
-   extensible at the edges.

EnvDoctor is **not** a secrets manager, deployment platform, hosted
service, environment injection tool, framework-specific configuration
library, or general static-analysis platform.

------------------------------------------------------------------------

## 2. Problem Definition

Typical configuration failures include:

-   required variable missing;
-   typo or unknown variable;
-   wrong type or format;
-   invalid enum value;
-   invalid numeric range;
-   stale `.env.example`;
-   local/CI drift;
-   undocumented configuration;
-   secret leakage in diagnostics;
-   requirements scattered across code and documentation.

The product thesis is:

> **Configuration should have a contract.**

The contract is the authoritative declaration of:

-   variables the application expects;
-   required vs optional variables;
-   semantic types;
-   constraints;
-   sensitivity;
-   documentation;
-   non-secret defaults.

------------------------------------------------------------------------

## 3. Product Positioning

### One-sentence positioning

> **EnvDoctor is a lightweight, language-agnostic configuration contract
> and validation CLI for application repositories.**

### Core promise

> **Define what your application expects. EnvDoctor tells you whether
> your environment satisfies it.**

### Not the product

EnvDoctor is not:

-   a secrets manager;
-   a hosted configuration service;
-   a secret synchronization platform;
-   a deployment manager;
-   a cloud dashboard;
-   an authentication/RBAC system;
-   a runtime environment injector in MVP;
-   a Docker orchestration tool;
-   a CI configuration parser;
-   a framework-specific environment library;
-   an AI configuration assistant;
-   a service-health checker;
-   a network reachability checker.

------------------------------------------------------------------------

## 4. Ecosystem Boundary

Adjacent tools already cover substantial parts of the space:

-   **dotenv-linter:** dotenv-file linting.
-   **dotenvx:** dotenv validation, loading, multiple environment files,
    encryption and runtime workflows.
-   **Runtime schema libraries:** typed validation inside application
    runtimes.
-   **Varlock / @env-spec:** repository-level environment schema,
    coercion, validation, sensitivity and environment workflows.

Therefore EnvDoctor must not differentiate merely by saying "we validate
`.env` files."

Its center of gravity is:

> **A source-independent repository configuration contract and
> diagnostic engine.**

The contract should be usable against different environment sources
without changing its semantics.

------------------------------------------------------------------------

## 5. Product Principles

1.  **Contract first** --- the contract is authoritative.
2.  **Local first** --- normal validation requires no network.
3.  **Language agnostic** --- no framework knowledge required.
4.  **Read-only by default** --- validation never mutates configuration.
5.  **Secrets are toxic data** --- values must never be casually
    exposed.
6.  **Deterministic** --- same inputs produce equivalent results on
    every OS.
7.  **Stable machine interface** --- diagnostic codes, exit codes and
    JSON are compatibility surfaces.
8.  **Small core** --- integrations belong at the edges.

------------------------------------------------------------------------

# 6. Contract Model

## 6.1 Definition

A contract is a versioned, declarative description of the configuration
variables an application expects, with constraints defining acceptable
values.

It does **not** define:

-   where secrets are stored;
-   how variables are injected;
-   how `.env` is loaded;
-   deployment infrastructure;
-   service connectivity;
-   runtime application behavior.

## 6.2 Canonical format

The canonical contract file is:

``` text
envdoctor.schema.json
```

It is based on **JSON Schema Draft 2020-12**, with a deliberately
restricted EnvDoctor profile.

EnvDoctor does **not** promise support for arbitrary JSON Schema.

## 6.3 Supported primitive types

-   `string`
-   `integer`
-   `number`
-   `boolean`

Environment variables are scalar at the boundary; `object`, `array`, and
`null` are not MVP variable types.

## 6.4 Supported variable constraints

-   `enum`
-   `const`
-   `default`
-   `description`
-   `title`
-   `pattern`
-   `minLength`
-   `maxLength`
-   `minimum`
-   `maximum`
-   `exclusiveMinimum`
-   `exclusiveMaximum`
-   `multipleOf`
-   `format`
-   `x-envdoctor-secret`

Supported formats:

-   `email`
-   `uri`

## 6.5 Requiredness

Requiredness uses the root `required` array.

Semantics:

  State       Required   Optional
  --------- ---------- ----------
  Missing        Error      Valid
  Empty          Error      Valid
  Present     Validate   Validate

A required variable must resolve to a non-empty value in MVP.

## 6.6 Defaults

`default` is descriptive metadata.

EnvDoctor never injects defaults.

If an optional variable is absent and has a default, validation passes
and EnvDoctor may state that a declared default exists.

A variable declared both `required` and with a `default` is a **contract
error**.

A secret variable with a default is also a **contract error**.

## 6.7 Enum

Enum matching is exact and case-sensitive after normal type
interpretation.

EnvDoctor must not silently normalize enum values.

## 6.8 Pattern

Pattern constraints use the regex semantics of the selected Go
implementation. Arbitrary executable validators are not supported.

## 6.9 Formats

`email` and `uri` are validation assertions in EnvDoctor.

A URI check is syntactic only. EnvDoctor never performs DNS, HTTP,
database, Redis, or credential checks.

## 6.10 Secret marker

``` json
"x-envdoctor-secret": true
```

marks a value as sensitive.

Secret status is independent of requiredness and type.

------------------------------------------------------------------------

# 7. Environment Value Semantics

Environment sources produce strings. EnvDoctor explicitly interprets
those strings.

## Integer

Accepted examples:

``` text
0
1
3000
-10
+10
0007
```

Rejected in MVP:

``` text
3.14
1e3
1_000
0x10
10ms
```

Surrounding whitespace is not silently trimmed.

## Number

Accepted:

``` text
0
3
3.14
-3.14
+3.14
.5
-.5
```

Rejected initially:

``` text
1e3
1_000
NaN
Infinity
```

## Boolean

Accepted, case-insensitively:

``` text
true
false
```

Rejected:

``` text
1
0
yes
no
on
off
enabled
disabled
```

## String

The parsed string is preserved; EnvDoctor does not silently trim or
otherwise normalize it.

------------------------------------------------------------------------

# 8. Unknown Variables

Unknown variables are warnings by default.

For example, `PATH`, `HOME`, `CI`, and other process variables should
not automatically become errors.

`ENV_UNKNOWN` is therefore a warning in the default policy.

A future strict policy may make selected warnings fatal, but MVP does
not make every warning an error.

------------------------------------------------------------------------

# 9. Contract Errors vs Environment Errors

These are separate classes.

### Contract error

The contract itself is invalid.

Examples:

-   unsupported keyword;
-   invalid contract version;
-   duplicate definition;
-   invalid `required`;
-   required + default;
-   secret + default;
-   invalid schema combination.

### Environment error

The contract is valid but the supplied environment violates it.

Examples:

-   missing required variable;
-   empty required variable;
-   invalid integer;
-   invalid boolean;
-   invalid enum;
-   pattern mismatch;
-   range violation;
-   invalid URI/email.

------------------------------------------------------------------------

# 10. Diagnostics

Every diagnostic contains at least:

``` text
severity
code
variable
message
```

Potential future metadata may include source/path/expected value, but no
secret value may appear.

Initial codes:

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

Codes are stable identifiers; messages can evolve.

------------------------------------------------------------------------

# 11. Validity and Exit Codes

An environment is valid when:

1.  the contract is valid;
2.  all required variables are present;
3.  required variables are non-empty;
4.  declared values satisfy their constraints;
5.  no error-severity diagnostic exists.

Warnings do not invalidate by default.

Initial exit codes:

``` text
0 = valid
1 = validation failure
2 = usage/contract/source error
3 = unexpected internal error
```

These codes are part of the CLI compatibility contract.

------------------------------------------------------------------------

# 12. Environment Sources

MVP supports:

1.  explicit dotenv file;
2.  current process environment.

The validator receives a normalized:

``` text
map[string]string
```

and does not care which source produced it.

MVP intentionally uses **one source per invocation**. There is no
`.env + .env.local + process environment` precedence model yet.

Future source composition requires an explicit precedence specification
before implementation.

------------------------------------------------------------------------

# 13. Dotenv Parsing

EnvDoctor must define its supported dotenv grammar explicitly.

MVP supports:

-   `KEY=value`;
-   blank lines;
-   comments;
-   quoted values;
-   single-quoted literals;
-   double-quoted values;
-   empty values;
-   documented escape behavior.

EnvDoctor does not execute shell commands or command substitution.

Interpolation is not a core EnvDoctor feature.

Unsupported syntax must produce a clear parser/source error rather than
silently producing a different value.

------------------------------------------------------------------------

# 14. Architecture

``` text
                         CLI
                          |
             +------------+------------+
             |                         |
       Contract Loader           Source Resolver
             |                         |
             |                  +------+------+
             |                  |             |
             |                .env        Process Env
             |                  |             |
             |                  +------+------+
             |                         |
             |                  Normalized Env
             |                         |
             +-------------+-----------+
                           |
                       Validator
                           |
                      Diagnostics
                      /          \
                Human Output   JSON Output

Contract
   |
Example Generator
   |
.env.example
```

Core validation must not depend on terminal rendering.

------------------------------------------------------------------------

# 15. CLI MVP

## `envdoctor init`

Creates an initial contract from existing repository configuration.

It may discover variable names but must never copy actual secret values
into the contract.

Inference is advisory, not authoritative.

## `envdoctor check`

Examples:

``` bash
envdoctor check
envdoctor check --env-file .env
envdoctor check --environment
```

It:

1.  loads the contract;
2.  validates the contract;
3.  loads the selected source;
4.  normalizes values;
5.  validates the environment;
6.  emits diagnostics;
7.  returns the appropriate exit code.

## `envdoctor generate example`

Generates `.env.example` from the contract.

Generation is deterministic and must not silently overwrite an existing
file.

## Not MVP

``` text
envdoctor diff
envdoctor audit
envdoctor doctor
envdoctor workspace
envdoctor run
```

------------------------------------------------------------------------

# 16. `.env.example` Rules

Only an explicit `default` may become a generated value.

Therefore:

-   enum without default → blank;
-   pattern without default → blank;
-   type without default → blank;
-   secret → blank;
-   actual environment values → never copied.

Generated ordering is deterministic.

------------------------------------------------------------------------

# 17. Security

Secret values must never appear in:

-   stdout;
-   stderr;
-   JSON output;
-   generated files;
-   diagnostics;
-   panic/error messages;
-   logs;
-   test snapshots;
-   telemetry.

There is no generic debug mode that prints all environment values.

Normal validation makes no network requests.

The Go implementation should minimize unnecessary copies of sensitive
strings where practical, but must not promise impossible
garbage-collector-level memory zeroization guarantees.

------------------------------------------------------------------------

# 18. Repository and Monorepo Scope

MVP is intentionally simple:

-   one invocation;
-   one contract;
-   one source.

Multiple applications can keep separate contracts:

``` text
apps/api/envdoctor.schema.json
apps/web/envdoctor.schema.json
apps/worker/envdoctor.schema.json
```

Schema imports, inheritance, composition and workspace graphs are not
MVP.

------------------------------------------------------------------------

# 19. Technical Direction

## Language

**Go is final.**

Primary reasons:

-   standalone binaries;
-   excellent cross-compilation;
-   no runtime dependency for users;
-   strong standard library;
-   fast CLI startup;
-   good fit for a local security-sensitive utility.

## Dependencies

Prefer the standard library.

Add dependencies only when they materially improve correctness or remove
significant complexity.

The application should begin as a single Go module.

Suggested conceptual packages:

``` text
cmd/
internal/
  contract/
  source/
  normalize/
  validate/
  diagnostic/
  generate/
  output/
```

Do not create speculative packages or separate modules.

------------------------------------------------------------------------

# 20. Testing

Tests must cover:

### Contract

-   valid/invalid contracts;
-   unsupported keywords;
-   version errors;
-   required/default conflicts;
-   secret/default conflicts;
-   invalid enum/type combinations.

### Dotenv

-   comments;
-   quoting;
-   empty values;
-   malformed lines;
-   duplicate keys;
-   unsupported interpolation;
-   CRLF/LF;
-   Unicode.

### Types

Every accepted/rejected integer, number and boolean representation.

### Validation

Every diagnostic code.

### Security

Assertions that secret values never appear in human output, JSON,
generated files, errors or logs.

### Cross-platform

Linux, Windows and macOS behavior.

------------------------------------------------------------------------

# 21. Documentation

Repository documentation must include:

``` text
README.md
LICENSE
CONTRIBUTING.md
SECURITY.md
CHANGELOG.md
docs/
```

Docs must cover:

-   installation;
-   quickstart;
-   contract reference;
-   dotenv grammar;
-   environment sources;
-   diagnostics;
-   exit codes;
-   security;
-   CI usage;
-   examples;
-   architecture;
-   contribution/release process.

------------------------------------------------------------------------

# 22. Closed Decisions

  Decision                 Final
  ------------------------ ---------------------------------------------
  Product                  EnvDoctor
  Purpose                  Repository environment contract validation
  Language                 Go
  Contract foundation      JSON Schema Draft 2020-12
  Contract profile         Restricted EnvDoctor subset
  Canonical file           `envdoctor.schema.json`
  Types                    string, integer, number, boolean
  Formats                  email, uri
  Secret marker            `x-envdoctor-secret`
  Required                 root `required`
  Defaults                 descriptive, never injected
  Unknown vars             warning by default
  Source model             source → normalized environment → validator
  MVP sources              dotenv file, process environment
  Source merging           no
  Profiles                 no
  Imports/inheritance      no
  Cross-variable logic     no
  Secrets manager          no
  Runtime injection        no
  Network/service checks   no
  Docker/CI analyzers      no
  AI                       no
  Hosted service           no

------------------------------------------------------------------------

# 23. References

-   JSON Schema 2020-12: https://json-schema.org/draft/2020-12
-   JSON Schema specification: https://json-schema.org/specification
-   dotenvx validation: https://dotenvx.com/docs/cli/validate/
-   dotenvx environment files: https://dotenvx.com/docs/env-file/
-   Varlock schema: https://varlock.dev/guides/schema/
-   Varlock environments: https://varlock.dev/guides/environments/
-   Go command documentation: https://go.dev/doc/cmd
-   Go cross-compilation: https://go.dev/wiki/WindowsCrossCompiling
