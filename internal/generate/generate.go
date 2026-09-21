// Package generate derives .env.example content from a configuration contract.
//
// Generation is one-directional and uses only contract metadata: actual
// environment values are never read, and no fake secret values are invented.
// Only an explicit non-secret default may become a generated value; every
// other variable is emitted blank. Output is deterministic and safe to re-parse
// with the EnvDoctor dotenv grammar.
package generate

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/haneefojay/envdoctor/internal/contract"
)

// envKeyPattern is the dotenv variable-name grammar. .env.example is a dotenv
// document, so every generated key must be expressible in that grammar.
var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Example renders the deterministic .env.example document for c. Declared
// variables appear in contract order (sorted by name). A variable with an
// explicit non-secret default gets that default as its value; secret variables
// and variables without a default are emitted blank. Example fails when a
// variable name cannot be expressed as a dotenv key.
func Example(c *contract.Contract) ([]byte, error) {
	if c == nil {
		return nil, errors.New("cannot generate an example from a nil contract")
	}

	var b strings.Builder
	for _, v := range c.Variables {
		if !envKeyPattern.MatchString(v.Name) {
			return nil, fmt.Errorf("variable %q cannot be expressed as a dotenv key", v.Name)
		}
		if v.Description != "" {
			writeDescription(&b, v.Description)
		}
		b.WriteString(v.Name)
		b.WriteByte('=')
		// Secrets never receive a value, not even a declared default: secret
		// values must never be written to a generated file.
		if !v.Secret && v.HasDefault() {
			b.WriteString(renderValue(v.Type, v.Default))
		}
		b.WriteByte('\n')
	}
	return []byte(b.String()), nil
}

// WriteExample writes content to path without overwriting an existing file.
// The non-overwrite check is atomic (O_EXCL), so there is no check-then-write
// window and an existing file is never silently replaced.
func WriteExample(path string, content []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// writeDescription writes a dotenv comment block. Descriptions are non-secret
// contract metadata; multiline descriptions become one comment line each.
func writeDescription(b *strings.Builder, text string) {
	for _, line := range strings.Split(text, "\n") {
		b.WriteString("# ")
		b.WriteString(strings.TrimSuffix(line, "\r"))
		b.WriteByte('\n')
	}
}

// renderValue formats an interpreted contract value as its dotenv text form.
// Type assertions are checked so a hand-constructed contract cannot cause a
// panic; a value that does not match the declared type renders as empty.
func renderValue(typ contract.Type, val contract.Value) string {
	switch typ {
	case contract.TypeString:
		s, ok := val.(string)
		if !ok {
			return ""
		}
		return renderString(s)
	case contract.TypeInteger:
		n, ok := val.(int64)
		if !ok {
			return ""
		}
		return strconv.FormatInt(n, 10)
	case contract.TypeNumber:
		f, ok := val.(float64)
		if !ok {
			return ""
		}
		// 'f' notation never uses exponents, so the result always satisfies the
		// project number grammar on reparse.
		return strconv.FormatFloat(f, 'f', -1, 64)
	case contract.TypeBoolean:
		b, ok := val.(bool)
		if !ok {
			return ""
		}
		if b {
			return "true"
		}
		return "false"
	}
	return ""
}

// renderString renders a string default so that re-parsing the generated file
// with the EnvDoctor dotenv grammar reproduces the identical value. Simple
// values are emitted verbatim; values that would be misinterpreted as quoted,
// spilled across lines, or read as interpolation are emitted double-quoted
// with the documented escape set.
func renderString(s string) string {
	if s == "" {
		return ""
	}
	if needsQuoting(s) {
		return quoteDouble(s)
	}
	return s
}

// needsQuoting reports whether emitting s as an unquoted value could change its
// meaning under the EnvDoctor dotenv grammar.
func needsQuoting(s string) bool {
	switch s[0] {
	case '\'', '"':
		return true
	}
	if strings.ContainsAny(s, "\r\n") {
		return true
	}
	return containsInterpolation(s)
}

// containsInterpolation mirrors the dotenv parser's detection of unsupported
// interpolation syntax: a '$' followed by a name-start character, '{', or '('.
func containsInterpolation(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '$' || i+1 >= len(s) {
			continue
		}
		switch next := s[i+1]; next {
		case '{', '(':
			return true
		default:
			if isNameStart(next) {
				return true
			}
		}
	}
	return false
}

// isNameStart reports whether c can begin a dotenv variable name.
func isNameStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// quoteDouble emits s as a double-quoted value using the documented escape set
// (\\, \", \$, \n, \r, \t). The parser reverses exactly these escapes, so the
// value round-trips byte-for-byte.
func quoteDouble(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '$':
			b.WriteString(`\$`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}
