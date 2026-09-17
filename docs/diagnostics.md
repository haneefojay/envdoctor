# Diagnostics

Every diagnostic contains `severity`, `code`, `variable`, and `message`.

Initial codes:

- Contract: `CONTRACT_INVALID`, `CONTRACT_UNSUPPORTED_KEYWORD`, `CONTRACT_DUPLICATE_VARIABLE`
- Environment: `ENV_MISSING`, `ENV_EMPTY`, `ENV_UNKNOWN`, `ENV_TYPE_MISMATCH`, `ENV_INVALID_ENUM`, `ENV_PATTERN_MISMATCH`, `ENV_NUMBER_OUT_OF_RANGE`, `ENV_INVALID_FORMAT`
- Sources: `ENV_FILE_INVALID`, `SOURCE_INVALID`

Exit codes are `0` for a valid environment, `1` for validation failure, `2` for usage/contract/source failure, and `3` for unexpected internal errors. Unknown variables are warnings by default.
