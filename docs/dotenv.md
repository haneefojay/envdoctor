# EnvDoctor Environment Sources

The MVP supports exactly two environment sources, both normalized into the
shared representation `map[string]string`:

- the process environment (the default for `envdoctor check`);
- one explicitly selected dotenv file.

Sources are never merged and have no precedence rules: `envdoctor check`
validates exactly one source per invocation. Values never cross the parsing
layer in any other form, which keeps the validator fully source-agnostic.

## Process environment

`envdoctor check` (without `--env-file`) validates the process environment.

- Names and values are preserved exactly: no case-normalization, no trimming.
- Names are semantically case-sensitive regardless of the host operating
  system.
- When the same key appears more than once, the last entry wins, which keeps
  behavior deterministic if a platform reports duplicates.
- Entries without a `=` separator are ignored.

## Dotenv grammar

`envdoctor check --env-file <path>` validates the file at `<path>`.

Supported syntax:

```text
KEY=value
KEY = value            # surrounding whitespace around the key is trimmed
KEY=                   # empty value; the key is present with value ""
KEY='literal'          # single quotes: fully literal, no escapes
KEY="double"           # double quotes: documented escapes below
# comment              # full-line comments (leading whitespace allowed)
                       # blank lines are ignored
```

Double-quoted values support exactly these escapes:

| Escape | Result          |
| ------ | --------------- |
| `\\`   | literal `\`     |
| `\"`   | literal `"`     |
| `\n`   | newline         |
| `\r`   | carriage return |
| `\t`   | tab             |
| `\$`   | literal `$`     |

Values are single-line. Line endings may be LF or CRLF; the trailing `\r` of a
CRLF line is never part of the value. File contents may be arbitrary UTF-8 and
are preserved byte-for-byte.

### Rejected inputs

Malformed input is a **source error**, never silently repaired:

- lines without a `KEY=VALUE` shape;
- a leading UTF-8 BOM (a byte-order mark before the first key) — a BOM from a
  Windows editor or `Save as UTF-8 with BOM` renderer is rejected as part of
  the first variable name, so save dotenv files as UTF-8 without BOM;
- variable names outside `[A-Za-z_][A-Za-z0-9_]*`;
- duplicate keys (never silently resolved; the last definition does not win);
- unterminated or unbalanced quoted values;
- trailing content after a quoted value;
- any escape outside the documented set;
- interpolation syntax.

### Interpolation

EnvDoctor does not support interpolation. The parser rejects `$name`, `${...}`,
and `$(...)` with a clear error ("interpolation is not supported; use single
quotes for a literal $ value") rather than producing a different value than the
author intended. Shell commands and command substitutions are never executed.
A bare `$` that does not begin interpolation syntax (for example `100$` or
`$5`) is a literal dollar sign.

### Values are preserved

Unquoted and double-quoted values are preserved verbatim, including
surrounding whitespace: values are never silently trimmed. An unquoted value
that contains `$` interpolation syntax is rejected. If you need a literal `$`
at the start of a value, use single quotes.

## Value semantics

After both sources normalize to `map[string]string`, values are interpreted
against their declared contract type. Strings pass through unchanged. Booleans
accept `true`/`false` case-insensitively only (`1`, `0`, `yes`, `no`, `on`,
`off`, and friends are rejected). Integers accept `0`, `1`, `3000`, `-10`,
`+10`, `0007` and reject `3.14`, `1e3`, `1_000`, `0x10`, `10ms`. Numbers accept
`0`, `3`, `3.14`, `-3.14`, `+3.14`, `.5`, `-.5` and reject `1e3`, `1_000`,
`NaN`, `Infinity`. See `internal/normalize` and the contract tests for the full
table.