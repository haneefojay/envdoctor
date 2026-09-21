// Package normalize converts environment-string values into the semantic
// values validated against a contract. It implements the fixed value grammar
// defined by the specification: strings are preserved verbatim, integers and
// numbers follow a deterministic digit grammar, and booleans accept only
// case-insensitive "true" and "false".
//
// Errors never embed the raw input: environment values can be secrets, and
// error text may surface in diagnostics.
//
// Normalization never trims, lowercases, or otherwise rewrites input before
// parsing. A value that is not a valid representation of its declared type
// fails, it is not silently coerced.
package normalize

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/haneefojay/envdoctor/internal/contract"
)

// Errors describing failed interpretations. Message text must never contain
// raw values.
var (
	ErrNotBoolean        = errors.New("not a valid boolean (accepted: true, false)")
	ErrNotInteger        = errors.New("not a valid integer")
	ErrIntegerOutOfRange = errors.New("integer is outside the representable range")
	ErrNotNumber         = errors.New("not a valid number")
	ErrNumberOutOfRange  = errors.New("number is outside the representable range")
	ErrUnsupportedType   = errors.New("unsupported variable type")
)

// integerGrammar matches an optional sign followed by one or more decimal
// digits. Leading zeros, "+10", "-10", and "0007" are accepted; exponents,
// separators, decimals, and suffixes are not. Surrounding whitespace is not
// trimmed, so " 5" does not match.
var integerGrammar = regexp.MustCompile(`^[+-]?[0-9]+$`)

// numberGrammar matches an optional sign followed by a decimal number. A
// fraction is required when a '.' is present. Exponents, separators, uppercase
// NaN/Infinity spellings, and bare punctuation do not match.
var numberGrammar = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]+)?|\.[0-9]+)$`)

// Interpret converts a raw environment string into the semantic value declared
// by typ: string, int64 for integer, float64 for number, or bool for boolean.
func Interpret(typ contract.Type, raw string) (contract.Value, error) {
	switch typ {
	case contract.TypeString:
		return String(raw), nil
	case contract.TypeInteger:
		return Integer(raw)
	case contract.TypeNumber:
		return Number(raw)
	case contract.TypeBoolean:
		return Boolean(raw)
	default:
		return nil, ErrUnsupportedType
	}
}

// String preserves the raw value verbatim. Strings are never trimmed.
func String(raw string) contract.Value {
	return raw
}

// Integer parses a signed decimal-integer string into an int64.
func Integer(raw string) (int64, error) {
	if !integerGrammar.MatchString(raw) {
		return 0, ErrNotInteger
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, ErrIntegerOutOfRange
	}
	return n, nil
}

// Number parses a signed decimal number (no exponent) into a float64.
func Number(raw string) (float64, error) {
	if !numberGrammar.MatchString(raw) {
		return 0, ErrNotNumber
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, ErrNumberOutOfRange
	}
	return f, nil
}

// Boolean parses "true" or "false" case-insensitively. No other value is
// accepted: 1, 0, yes, no, on, off, enabled, disabled are all rejected.
func Boolean(raw string) (bool, error) {
	switch strings.ToLower(raw) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, ErrNotBoolean
	}
}
