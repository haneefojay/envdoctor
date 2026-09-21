# EnvDoctor Diagnostics

Diagnostics are the structured, stable output of a validation run. A
diagnostic always has four fields:

| Field      | Meaning                                                              |
| ---------- | -------------------------------------------------------------------- |
| `severity` | `error` or `warning`.                                                |
| `code`     | A stable, API-level identifier.                                      |
| `variable` | The variable the diagnostic concerns, or `""` for contract- and source-level findings. |
| `message`  | A human-readable explanation. Messages may evolve; codes do not.     |

Messages never contain environment values: values can be secrets. See
`docs/security.md`.

## Diagnostic codes

| Code                          | Severity | Meaning |
| ----------------------------- | -------- | ------- |
| `CONTRACT_INVALID`            | error    | The contract could not be parsed (malformed JSON, unsupported `$schema`, invalid root shape, contradictory constraints, defaults violating constraints, ...). |
| `CONTRACT_UNSUPPORTED_KEYWORD`| error    | The contract uses a keyword outside the EnvDoctor Contract Profile. |
| `CONTRACT_DUPLICATE_VARIABLE` | error    | A variable is defined or required more than once. |
| `ENV_MISSING`                 | error    | A required variable is absent. |
| `ENV_EMPTY`                   | error    | A required variable is present but empty. |
| `ENV_UNKNOWN`                 | warning  | A variable in the environment is not declared in the contract. |
| `ENV_TYPE_MISMATCH`           | error    | The value is not a valid representation of its declared type. |
| `ENV_INVALID_ENUM`            | error    | The value is not a member of the declared `enum`, or does not match the declared `const`. |
| `ENV_PATTERN_MISMATCH`        | error    | The value does not match the declared `pattern`. |
| `ENV_NUMBER_OUT_OF_RANGE`     | error    | The value violates a `minimum`/`maximum`/`exclusiveMinimum`/`exclusiveMaximum` bound. |
| `ENV_INVALID_FORMAT`          | error    | The value does not satisfy the declared `format` (`email`/`uri`), syntactically. |
| `ENV_LENGTH_OUT_OF_RANGE`     | error    | The value length violates `minLength`/`maxLength` (measured in UTF-8 runes). |
| `ENV_MULTIPLE_OF`             | error    | The value is not a whole-number multiple of the declared `multipleOf`. |
| `ENV_FILE_INVALID`            | error    | The selected env file is malformed dotenv (a source error). |
| `SOURCE_INVALID`              | error    | The selected env file could not be read (a source error). |

`CONTRACT_*` and source-level codes carry an empty `variable` field.

## Ordering

Deterministic ordering is a product invariant; nothing depends on Go map
iteration order.

- The validator emits, per declared variable (sorted by name), then unknown
  variables (sorted by name).
- The output layer canonicalizes diagnostics into the final order: errors
  before warnings, then by variable name, code, and message.

## Human output

`envdoctor check` (without `--json`) writes a terminal-friendly report:

```text
Environment validation failed

✗ DATABASE_URL
  ERROR ENV_MISSING
  missing required variable

✗ PORT
  ERROR ENV_TYPE_MISMATCH
  value is not a valid integer

⚠ EXTRA_SETTING
  WARN ENV_UNKNOWN
  unknown environment variable

2 errors, 1 warning
```

- An environment with no diagnostics prints `Environment is valid`.
- Warnings without errors print `Environment is valid, with warnings`.
- The summary line reports `N errors, N warnings`.
- Contract- and source-level diagnostics have no `variable`, so the message is
  shown as the subject line and the fourth line is omitted.
- Output contains no ANSI escape sequences, uses only LF line endings, and
  never includes environment values.

## JSON output

`envdoctor check --json` writes the stable machine-readable document:

```json
{
  "valid": false,
  "diagnostics": [
    { "severity": "error", "code": "ENV_MISSING", "variable": "DATABASE_URL", "message": "missing required variable" }
  ]
}
```

- The shape is always `{ "valid": bool, "diagnostics": [...] }`; the array is
  always present.
- Indented with two spaces; valid JSON; contains no ANSI sequences; uses LF
  line endings.
- `valid` is `false` exactly when at least one error-severity diagnostic is
  present; warnings do not invalidate an environment.

## Exit codes

| Code | Meaning                              |
| ---- | ------------------------------------ |
| 0    | Environment is valid.                |
| 1    | Validation failure (`valid: false`). |
| 2    | Usage, contract, or source error.    |
| 3    | Unexpected internal error.           |

A contract or source error reported through `envdoctor check` is still rendered
in the same human or JSON format but exits with code 2.

## What makes an environment valid

An environment is valid only when:

- the contract is valid;
- required variables are present and non-empty;
- values satisfy their declared constraints;
- there are no error-severity diagnostics.

Unknown-variable warnings do not invalidate an environment.