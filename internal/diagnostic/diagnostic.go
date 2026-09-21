// Package diagnostic defines the stable structured diagnostics produced by
// EnvDoctor. Diagnostics are presentation-independent: output packages render
// them, and the validation engine produces them without knowing about the
// terminal or the environment source.
//
// Diagnostic codes are stable API-level identifiers. Messages may evolve, but
// codes and the severity/variable/message shape are compatibility surfaces.
// Secrets must never appear in any Diagnostic field.
package diagnostic

import "fmt"

// Severity classifies a diagnostic.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Code is a stable diagnostic identifier.
type Code string

// Stable diagnostic codes. These are compatibility surfaces; do not rename.
const (
	CodeContractInvalid            Code = "CONTRACT_INVALID"
	CodeContractUnsupportedKeyword Code = "CONTRACT_UNSUPPORTED_KEYWORD"
	CodeContractDuplicateVariable  Code = "CONTRACT_DUPLICATE_VARIABLE"

	CodeEnvMissing          Code = "ENV_MISSING"
	CodeEnvEmpty            Code = "ENV_EMPTY"
	CodeEnvUnknown          Code = "ENV_UNKNOWN"
	CodeEnvTypeMismatch     Code = "ENV_TYPE_MISMATCH"
	CodeEnvInvalidEnum      Code = "ENV_INVALID_ENUM"
	CodeEnvPatternMismatch  Code = "ENV_PATTERN_MISMATCH"
	CodeEnvNumberOutOfRange Code = "ENV_NUMBER_OUT_OF_RANGE"
	CodeEnvInvalidFormat    Code = "ENV_INVALID_FORMAT"

	// CodeEnvLengthOutOfRange reports a minLength/maxLength violation. These
	// keywords are supported by the contract profile, and this code extends the
	// initial diagnostic-code list ("Initial codes include" is not exhaustive).
	CodeEnvLengthOutOfRange Code = "ENV_LENGTH_OUT_OF_RANGE"

	// CodeEnvMultipleOf reports a multipleOf violation for the supported
	// multipleOf keyword (see CodeEnvLengthOutOfRange note).
	CodeEnvMultipleOf Code = "ENV_MULTIPLE_OF"

	CodeEnvFileInvalid Code = "ENV_FILE_INVALID"
	CodeSourceInvalid  Code = "SOURCE_INVALID"
)

// Diagnostic is a single structured finding.
//
// Variable names the configuration variable the diagnostic concerns, or is an
// empty string for contract- or source-level findings that do not map to a
// single variable. Message must never contain secret values.
type Diagnostic struct {
	Severity Severity `json:"severity"`
	Code     Code     `json:"code"`
	Variable string   `json:"variable"`
	Message  string   `json:"message"`
}

// Errorf creates an error-severity diagnostic. The message is formatted with
// fmt.Sprintf and must never embed secret values.
func Errorf(code Code, variable, format string, args ...any) Diagnostic {
	return Diagnostic{
		Severity: SeverityError,
		Code:     code,
		Variable: variable,
		Message:  fmt.Sprintf(format, args...),
	}
}

// Warningf creates a warning-severity diagnostic. The message is formatted
// with fmt.Sprintf and must never embed secret values.
func Warningf(code Code, variable, format string, args ...any) Diagnostic {
	return Diagnostic{
		Severity: SeverityWarning,
		Code:     code,
		Variable: variable,
		Message:  fmt.Sprintf(format, args...),
	}
}
