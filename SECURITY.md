# Security Policy

## Reporting a Vulnerability

Do not open an issue. Report suspected vulnerabilities privately to the
maintainers via a GitHub Security Advisory or a private email to the repository
owner.

Please include a description of the issue, any proof of concept, and the
EnvDoctor version affected. Reproduce against the latest `main` where possible.

## Security Promise

EnvDoctor handles environment values that may be secrets.

The product promise: **actual secret values never appear in output.** This
means secrets never appear in:

- stdout or stderr;
- human diagnostics;
- JSON output;
- generated files such as `.env.example`;
- panic or error messages;
- logs;
- test snapshots.

This is enforced by design and tested: security tests deliberately use realistic
secret values and assert their absence from every user-visible representation.

Normal validation performs no network requests, executes no shell commands, and
does not use telemetry.

## Scope

EnvDoctor is not a secrets manager, encryption tool, or configuration service.
It never stores, encrypts, or synchronizes secret values.

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| main    | yes (unreleased)   |
| < 0.1   | no                 |