# EnvDoctor Quickstart

EnvDoctor defines the configuration contract between an application and its
environment, validates an environment against that contract, and explains
failures without exposing secrets.

**Prerequisites:** Go 1.27 or newer. EnvDoctor is a single standalone binary
with no runtime dependencies.

## Build

```text
go build -o envdoctor ./cmd/envdoctor
```

This builds the standalone binary as `envdoctor` in the current directory. You
can also install it on your PATH:

```text
go install github.com/haneefojay/envdoctor/cmd/envdoctor
```

## 1. Create an initial contract

From the repository root, run:

```text
envdoctor init
```

`init` is an adoption assistant, not an oracle. It discovers variable names
from `.env.example` (preferred) or `.env`, **copies no values**, and emits
`envdoctor.schema.json` with every discovered variable declared as an optional
string. You are expected to review and edit it before trusting it:

```text
Wrote envdoctor.schema.json with 5 variables discovered from .env.example.
Review the draft before running envdoctor check: adjust types, mark secrets,
and set required variables.
```

`init` refuses to overwrite an existing file.

## 2. Write or refine the contract

A contract is a restricted JSON Schema Draft 2020-12 document. Minimal example:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "title": "Example API service",
  "description": "Configuration contract for the example service.",
  "required": ["DATABASE_URL", "PORT"],
  "properties": {
    "DATABASE_URL": {
      "type": "string",
      "format": "uri",
      "x-envdoctor-secret": true,
      "description": "PostgreSQL connection string."
    },
    "PORT": {
      "type": "integer",
      "minimum": 1,
      "maximum": 65535,
      "default": 3000
    },
    "LOG_LEVEL": {
      "type": "string",
      "enum": ["debug", "info", "warn", "error"],
      "default": "info"
    }
  }
}
```

See `docs/contract.md` for the full reference.

## 3. Validate the environment

The contract is discovered as `envdoctor.schema.json` in the current directory.
Validate the process environment (the default):

```text
envdoctor check
```

Validate a dotenv file explicitly:

```text
envdoctor check --env-file .env
```

Validate another contract file:

```text
envdoctor check --contract path/to/schema.json --env-file .env
```

The two sources are never merged: choose `--env-file` or `--environment`
(default), not both.

### Exit codes

| Code | Meaning                                        |
| ---- | ---------------------------------------------- |
| 0    | Environment is valid (warnings are allowed)    |
| 1    | Validation failure (one or more errors)        |
| 2    | Usage, contract, or source error               |
| 3    | Unexpected internal error                      |

### Human output

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

### Machine output

```text
envdoctor check --json
```

```json
{
  "valid": false,
  "diagnostics": [
    {
      "severity": "error",
      "code": "ENV_MISSING",
      "variable": "DATABASE_URL",
      "message": "missing required variable"
    }
  ]
}
```

The JSON document is always `{ "valid": bool, "diagnostics": [...] }`, is
indented with two spaces, and contains no ANSI sequences. See
`docs/diagnostics.md`.

## 4. Generate `.env.example`

```text
envdoctor generate example
```

The example file is derived from the contract only: an explicit non-secret
default becomes the generated value; secrets and variables without a default
are emitted blank. Existing files are never overwritten. See `docs/contract.md`.

## Also see

- `docs/contract.md` - the contract language and semantics
- `docs/dotenv.md` - the dotenv source grammar
- `docs/diagnostics.md` - codes, ordering, and exit codes
- `docs/security.md` - how secrets are protected
- `docs/ci.md` - how CI runs EnvDoctor's checks