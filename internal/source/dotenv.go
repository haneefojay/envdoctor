package source

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// ParseError describes a malformed dotenv document. Line is 1-based, or 0 when
// the error is not tied to a single line.
type ParseError struct {
	Line int
	Msg  string
}

func (e *ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("line %d: %s", e.Line, e.Msg)
	}
	return e.Msg
}

func parseError(line int, format string, args ...any) error {
	return &ParseError{Line: line, Msg: fmt.Sprintf(format, args...)}
}

// LoadDotenv reads the file at path and parses it as a dotenv document.
// A read failure returns the underlying OS error; malformed content returns a
// *ParseError.
func LoadDotenv(path string) (Environment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseDotenv(data)
}

// ParseDotenv parses a dotenv document into an Environment.
//
// Supported syntax:
//
//	KEY=value
//	KEY = value           (surrounding whitespace around the key is trimmed)
//	KEY=                  (empty value; present in the Environment)
//	KEY='literal'         (single quotes: fully literal, no escapes)
//	KEY="double"          (double quotes: documented escapes below)
//	# comment             (leading-whitespace line comments)
//	blank lines
//
// Double-quoted values support the escapes \\, \", \n, \r, \t, and \$; any
// other escape is an error. Values are single-line.
//
// Rejected with a ParseError:
//
//   - lines without a KEY=VALUE shape;
//   - variable names outside [A-Za-z_][A-Za-z0-9_]*;
//   - duplicate keys (never silently resolved);
//   - unterminated or unbalanced quoted values;
//   - trailing content after a quoted value;
//   - unsupported escapes;
//   - interpolation syntax ($var, ${var}, $(...)), which EnvDoctor does not
//     support. Single-quoted values are the documented way to carry a literal
//     dollar sign.
//
// Unquoted and double-quoted values are otherwise preserved verbatim,
// including surrounding whitespace: values are never silently trimmed.
// Interpolation is never evaluated; shell commands are never executed.
func ParseDotenv(data []byte) (Environment, error) {
	env := make(Environment)
	seen := make(map[string]bool)
	for i, rawLine := range strings.Split(string(data), "\n") {
		lineNo := i + 1
		// CRLF: the trailing '\r' of a line ending is not part of the value.
		line := strings.TrimSuffix(rawLine, "\r")

		trimmed := strings.Trim(line, " \t")
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		key, err := parseKeyLine(line, lineNo)
		if err != nil {
			return nil, err
		}

		if seen[key] {
			return nil, parseError(lineNo, "variable %q appears more than once", key)
		}
		seen[key] = true

		eq := strings.IndexByte(line, '=')
		raw := line[eq+1:]
		value, err := parseValue(raw, lineNo)
		if err != nil {
			return nil, err
		}
		env[key] = value
	}
	return env, nil
}

// parseKeyLine validates the key half of a KEY=VALUE line and returns the
// validated key. The '=' is guaranteed to exist.
func parseKeyLine(line string, lineNo int) (string, error) {
	eq := strings.IndexByte(line, '=')
	if eq < 0 {
		return "", parseError(lineNo, "expected KEY=VALUE but found no %q", "=")
	}
	keyPart := strings.Trim(line[:eq], " \t")
	if !isValidKey(keyPart) {
		return "", parseError(lineNo, "invalid variable name %q; expected [A-Za-z_][A-Za-z0-9_]*", keyPart)
	}
	return keyPart, nil
}

// isValidKey reports whether the key matches [A-Za-z_][A-Za-z0-9_]*.
func isValidKey(k string) bool {
	if k == "" {
		return false
	}
	for i := 0; i < len(k); i++ {
		c := k[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c == '_':
		default:
			if i == 0 || c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

// parseValue interprets the value half of a KEY=VALUE line.
func parseValue(raw string, lineNo int) (string, error) {
	if raw == "" {
		return "", nil
	}
	switch raw[0] {
	case '\'':
		return parseSingleQuoted(raw, lineNo)
	case '"':
		return parseDoubleQuoted(raw, lineNo)
	default:
		return parseUnquoted(raw, lineNo)
	}
}

// parseSingleQuoted parses a '...' literal. Single-quoted values are fully
// literal: no escapes and no interpolation processing.
func parseSingleQuoted(raw string, lineNo int) (string, error) {
	if len(raw) < 2 || raw[len(raw)-1] != '\'' {
		return "", parseError(lineNo, "unterminated single-quoted value")
	}
	inner := raw[1 : len(raw)-1]
	if strings.ContainsRune(inner, '\'') {
		return "", parseError(lineNo, "unbalanced single-quoted value")
	}
	return inner, nil
}

// parseDoubleQuoted parses a "..." value with the documented escape set.
func parseDoubleQuoted(raw string, lineNo int) (string, error) {
	if len(raw) < 2 || raw[len(raw)-1] != '"' {
		return "", parseError(lineNo, "unterminated double-quoted value")
	}
	var b strings.Builder
	inner := raw[1 : len(raw)-1]
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		switch c {
		case '\\':
			if i+1 >= len(inner) {
				return "", parseError(lineNo, "dangling escape at end of double-quoted value")
			}
			i++
			esc := inner[i]
			switch esc {
			case '\\', '"', 'n', 'r', 't', '$':
				b.WriteByte(docEscaped(esc))
			default:
				return "", parseError(lineNo, "unsupported escape \\%c in double-quoted value", esc)
			}
		case '$':
			if nextIsInterpolation(inner, i) {
				return "", parseError(lineNo, "interpolation is not supported; use single quotes for a literal $ value")
			}
			b.WriteByte('$')
		default:
			b.WriteByte(c)
		}
	}
	return b.String(), nil
}

// docEscaped maps a documented escape character to its resulting byte.
func docEscaped(esc byte) byte {
	switch esc {
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	default:
		return esc // \\, \", \$
	}
}

// parseUnquoted preserves the value verbatim. Interpolation syntax is rejected
// rather than silently leaving a different value than the author intended.
func parseUnquoted(raw string, lineNo int) (string, error) {
	for i := 0; i < len(raw); i++ {
		if raw[i] == '$' && nextIsInterpolation(raw, i) {
			return "", parseError(lineNo, "interpolation is not supported; use single quotes for a literal $ value")
		}
	}
	return raw, nil
}

// nextIsInterpolation reports whether the '$' at position i begins unsupported
// interpolation syntax: ${...} (JSON-like or shell braces), $(...) command
// substitution, or a $name variable reference. A bare $ followed by a
// non-name character (for example "100$" or "$5") is a literal dollar sign.
func nextIsInterpolation(s string, i int) bool {
	if i+1 >= len(s) {
		return false
	}
	next := s[i+1]
	switch next {
	case '{', '(':
		return true
	}
	return isNameStart(next)
}

func isNameStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

// IsParseError reports whether err is a dotenv parse error (as opposed to a
// file-read or other source error).
func IsParseError(err error) bool {
	var pe *ParseError
	return errors.As(err, &pe)
}
