# Architecture

EnvDoctor follows a source-independent validation pipeline:

```text
contract file -> normalized contract
                         +
dotenv/process environment -> normalized string map
                         |
                    validator
                         |
                    diagnostics
                         |
                 human or JSON output
```

The MVP keeps the CLI read-only by default, uses one environment source per invocation, sorts externally visible results deterministically, and treats the contract as authoritative. Secret values are never included in diagnostics or generated files.
