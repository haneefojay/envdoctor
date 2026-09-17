# Security Policy

EnvDoctor treats environment values as sensitive data.

- Secret values must never appear in stdout, stderr, diagnostics, JSON, logs, errors, generated files, or tests.
- Normal validation makes no network requests.
- Dotenv parsing does not execute shell commands or command substitutions.
- Unknown variables are warnings by default.

Please report suspected vulnerabilities privately through GitHub's security reporting channel.
