# EnvDoctor Contract Reference

The canonical contract file is `envdoctor.schema.json`. It uses JSON Schema
Draft 2020-12, but EnvDoctor supports a deliberately restricted profile: the
**EnvDoctor Contract Profile**. Arbitrary JSON Schema is never accepted;
unsupported or invalid constructs produce a contract error.

## Document shape

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["DATABASE_URL"],
  "properties": {
    "DATABASE_URL": { "type": "string" }
  }
}
```

### Root-level fields

| Field         | Value / meaning                                                            |
| ------------- | -------------------------------------------------------------------------- |
| `$schema`     | Must be exactly `https://json-schema.org/draft/2020-12/schema`. This is the contract version gate; any other or absent `$schema` is a contract error. |
| `type`        | Must be `object`.                                                          |
| `title`       | Optional document title (descriptive metadata).                            |
| `description` | Optional document description (descriptive metadata).                      |
| `required`    | Array of required variable names (see Requiredness).                       |
| `properties`  | Object mapping variable names to variable schemas.                         |

The document must be a single JSON object: malformed JSON and trailing content
after the root object are contract errors. Duplicate JSON keys are rejected
rather than silently resolved.

Variable names follow the dotenv key grammar `[A-Za-z_][A-Za-z0-9_]*`; a
variable that cannot be expressed in the dotenv grammar cannot be round-tripped
and is rejected when generating an example.

## Variable types

Supported environment-variable primitive types:

| Type      | Stored as | Notes                                            |
| --------- | --------- | ------------------------------------------------ |
| `string`  | `string`  | Preserved verbatim; never trimmed.               |
| `integer` | `int64`   | See `docs/dotenv.md` for the accepted grammar.    |
| `number`  | `float64` | See `docs/dotenv.md` for the accepted grammar.    |
| `boolean` | `bool`    | Accepts `true`/`false` case-insensitively only.   |

`object`, `array`, and `null` variable types are not supported in the MVP and
produce a contract error. Every variable must declare exactly one of the four
supported types: a missing `type` (even an empty variable schema `{}`) is a
contract error.

## Variable-level keywords

| Keyword             | Applies to | Meaning |
| ------------------- | ---------- | ------- |
| `type`              | all        | One of `string`, `integer`, `number`, `boolean`. |
| `title`             | all        | Optional short label (metadata). |
| `description`       | all        | Optional explanation (metadata); may become a comment in generated `.env.example`. |
| `default`           | all        | Descriptive metadata only; EnvDoctor never injects defaults into the process environment. The application remains responsible for applying runtime defaults. |
| `enum`              | all        | Exact, case-sensitive membership check after type interpretation. |
| `const`             | all        | Exact, case-sensitive equality check after type interpretation; must be a member of `enum` when both are present. |
| `pattern`           | `string`   | Go regexp; the value must match. |
| `minLength`         | `string`   | Minimum value length in UTF-8 runes. |
| `maxLength`         | `string`   | Maximum value length in UTF-8 runes. |
| `minimum`           | `integer`, `number` | Inclusive lower bound. |
| `maximum`           | `integer`, `number` | Inclusive upper bound. |
| `exclusiveMinimum`  | `integer`, `number` | Exclusive lower bound. |
| `exclusiveMaximum`  | `integer`, `number` | Exclusive upper bound. |
| `multipleOf`        | `integer`, `number` | Value must be a whole-number multiple. |
| `format`            | `string`   | `email` or `uri`; syntactic validation only, never a network or DNS check. |
| `x-envdoctor-secret` | all       | `true` marks the variable as secret; see `docs/security.md`. |

An empty struct `{}` in `properties` declares a metadata-only variable whose
type is unconstrained; it requires no value checks beyond presence and
non-emptiness for required variables.

## Requiredness

Requiredness is expressed with the root `required` array. A required variable:

- must exist in the environment;
- must be non-empty;
- must satisfy its declared constraints.

An optional variable may be absent. Unknown variables present in the
environment are warnings, never errors. See `docs/diagnostics.md`.

## Contract errors

The following produce contract diagnostics (exit code 2 from `envdoctor check`):

- unsupported or unknown `$schema` / missing `$schema`;
- root `type` other than `object`;
- unsupported root or variable keywords;
- duplicate variable definitions (a property declared twice) or a variable
  listed more than once in `required`;
- a required variable that declares a default;
- a secret variable that declares a default;
- invalid or contradictory constraints: for example `minLength` greater than
  `maxLength`, `minimum` greater than `maximum`, exclusive bounds that no value
  can satisfy, or a `const` that is not a member of its `enum`;
- constraints attached to the wrong type: numeric bounds on `string` or
  `boolean`, length constraints on `boolean`, or `pattern`/`format` on a
  non-string variable;
- defaults that violate their own constraints;
- a `pattern` that does not compile;
- malformed JSON, trailing content after the root object, or duplicate JSON
  keys;
- environment-variable types other than `string`, `integer`, `number`,
  `boolean`.

## Defaults and example generation

`envdoctor generate example` derives `.env.example` from the contract only:

- explicit default -> the generated value may use the default;
- secret -> blank (never a value, never a fake);
- no default -> blank, regardless of `enum`, `pattern`, or `type`.

`init` never copies environment values and never infers authoritative types,
requiredness, defaults, or secret classification; discovered names are emitted
as optional strings.