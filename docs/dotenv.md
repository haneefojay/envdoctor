# Dotenv Grammar

Supported syntax includes `KEY=value`, blank lines, comments, empty values, single-quoted literals, double-quoted values, UTF-8, LF, and CRLF line endings.

Duplicate keys, malformed assignments, interpolation, shell commands, and unsupported quoting are rejected. Values are not silently trimmed. Each invocation uses one explicit dotenv file or the process environment; sources are not merged.
