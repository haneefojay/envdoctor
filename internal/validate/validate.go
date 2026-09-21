// Package validate evaluates a normalized environment against a contract and
// produces structured diagnostics. It is presentation- and source-agnostic:
// it consumes a normalized map[string]string and a parsed contract and knows
// nothing about terminals, dotenv parsing, or output rendering.
//
// Value interpretation follows the fixed grammar in internal/normalize.
// Diagnostic messages never embed environment values: values can be secrets.
package validate

import (
	"math"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/haneefojay/envdoctor/internal/contract"
	"github.com/haneefojay/envdoctor/internal/diagnostic"
	"github.com/haneefojay/envdoctor/internal/normalize"
	"github.com/haneefojay/envdoctor/internal/source"
)

// emailFormat matches a pragmatic, syntactic-only email shape: a local part, a
// single @, and a dot-separated domain. No network or DNS checks are ever run.
var emailFormat = regexp.MustCompile(`^[A-Za-z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)*$`)

// Validate evaluates env against the contract and returns diagnostics in
// deterministic order: per declared variable (sorted by name), then unknown
// variables (sorted by name). A nil contract is itself a contract error.
func Validate(c *contract.Contract, env source.Environment) []diagnostic.Diagnostic {
	if c == nil {
		return []diagnostic.Diagnostic{
			diagnostic.Errorf(diagnostic.CodeContractInvalid, "", "no contract supplied"),
		}
	}

	declared := make(map[string]bool, len(c.Variables))
	var out []diagnostic.Diagnostic

	for _, v := range c.Variables {
		declared[v.Name] = true
		raw, ok := env[v.Name]
		if !ok {
			if v.Required {
				out = append(out, diagnostic.Errorf(diagnostic.CodeEnvMissing, v.Name, "missing required variable"))
			}
			continue
		}
		if v.Required && raw == "" {
			out = append(out, diagnostic.Errorf(diagnostic.CodeEnvEmpty, v.Name, "required variable has an empty value"))
			continue
		}
		out = append(out, validateValue(v, raw)...)
	}

	var unknown []string
	for key := range env {
		if !declared[key] {
			unknown = append(unknown, key)
		}
	}
	sort.Strings(unknown)
	for _, key := range unknown {
		out = append(out, diagnostic.Warningf(diagnostic.CodeEnvUnknown, key, "unknown environment variable"))
	}

	return out
}

// validateValue validates a single non-missing, non-empty value. An optional
// variable that is present with an empty value is validated normally: the empty
// string is a legitimate value for some types, and invalid for others.
func validateValue(v *contract.Variable, raw string) []diagnostic.Diagnostic {
	val, err := normalize.Interpret(v.Type, raw)
	if err != nil {
		return []diagnostic.Diagnostic{
			diagnostic.Errorf(diagnostic.CodeEnvTypeMismatch, v.Name, "value is not a valid %s", v.Type),
		}
	}

	var out []diagnostic.Diagnostic

	if len(v.Enum) > 0 && !enumContains(v.Enum, val) {
		out = append(out, diagnostic.Errorf(diagnostic.CodeEnvInvalidEnum, v.Name, "value is not one of the allowed values"))
	}
	if v.HasConst() && !equalValues(v.Const, val) {
		out = append(out, diagnostic.Errorf(diagnostic.CodeEnvInvalidEnum, v.Name, "value does not match the declared const"))
	}

	switch v.Type {
	case contract.TypeString:
		s := val.(string)
		if v.Pattern != nil && !v.Pattern.MatchString(s) {
			out = append(out, diagnostic.Errorf(diagnostic.CodeEnvPatternMismatch, v.Name, "value does not match pattern /%s/", v.PatternRaw))
		}
		if n := utf8.RuneCountInString(s); v.MinLength != nil && n < *v.MinLength {
			out = append(out, diagnostic.Errorf(diagnostic.CodeEnvLengthOutOfRange, v.Name, "value length %d is below the minimum of %d", n, *v.MinLength))
		}
		if n := utf8.RuneCountInString(s); v.MaxLength != nil && n > *v.MaxLength {
			out = append(out, diagnostic.Errorf(diagnostic.CodeEnvLengthOutOfRange, v.Name, "value length %d exceeds the maximum of %d", n, *v.MaxLength))
		}
		if v.Format != "" && !validFormat(v.Format, s) {
			out = append(out, diagnostic.Errorf(diagnostic.CodeEnvInvalidFormat, v.Name, "value does not have a valid %s format", v.Format))
		}

	case contract.TypeInteger, contract.TypeNumber:
		f := toFloat(val)
		if v.Minimum != nil && f < *v.Minimum {
			out = append(out, diagnostic.Errorf(diagnostic.CodeEnvNumberOutOfRange, v.Name, "value is below the minimum %g", *v.Minimum))
		}
		if v.Maximum != nil && f > *v.Maximum {
			out = append(out, diagnostic.Errorf(diagnostic.CodeEnvNumberOutOfRange, v.Name, "value is above the maximum %g", *v.Maximum))
		}
		if v.ExclusiveMinimum != nil && f <= *v.ExclusiveMinimum {
			out = append(out, diagnostic.Errorf(diagnostic.CodeEnvNumberOutOfRange, v.Name, "value must be greater than %g", *v.ExclusiveMinimum))
		}
		if v.ExclusiveMaximum != nil && f >= *v.ExclusiveMaximum {
			out = append(out, diagnostic.Errorf(diagnostic.CodeEnvNumberOutOfRange, v.Name, "value must be less than %g", *v.ExclusiveMaximum))
		}
		if v.MultipleOf != nil && !isMultipleOf(f, *v.MultipleOf) {
			out = append(out, diagnostic.Errorf(diagnostic.CodeEnvMultipleOf, v.Name, "value must be a multiple of %g", *v.MultipleOf))
		}
	}

	return out
}

// validFormat applies the syntactic format validations. URI validation uses
// net/url; it is never a network or DNS check.
func validFormat(format, s string) bool {
	switch format {
	case contract.FormatEmail:
		return emailFormat.MatchString(s)
	case contract.FormatURI:
		if strings.ContainsAny(s, " \t\r\n") {
			return false
		}
		u, err := url.Parse(s)
		return err == nil && u.Scheme != ""
	default:
		return true
	}
}

// enumContains reports whether val exactly matches one of the interpreted enum
// values. String matching is exact and case-sensitive.
func enumContains(enum []contract.Value, val contract.Value) bool {
	for _, e := range enum {
		if equalValues(e, val) {
			return true
		}
	}
	return false
}

// equalValues compares two interpreted scalar values. Compatible numeric
// representations are compared via type switch, never through reflection.
func equalValues(a, b contract.Value) bool {
	switch x := a.(type) {
	case string:
		y, ok := b.(string)
		return ok && x == y
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case int64:
		y, ok := b.(int64)
		return ok && x == y
	case float64:
		y, ok := b.(float64)
		return ok && x == y
	}
	return false
}

func toFloat(v contract.Value) float64 {
	switch x := v.(type) {
	case int64:
		return float64(x)
	case float64:
		return x
	}
	return 0
}

// isMultipleOf reports whether f is a whole-number multiple of m, tolerating
// floating-point rounding for decimal multiples such as 0.25.
func isMultipleOf(f, m float64) bool {
	if m == 0 {
		return false
	}
	q := f / m
	return math.Abs(q-math.Round(q)) <= 1e-9
}
