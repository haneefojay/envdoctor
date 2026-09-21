package generate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/haneefojay/envdoctor/internal/contract"
	"github.com/haneefojay/envdoctor/internal/source"
)

// parseContract builds a Contract from contract JSON, failing the test on any
// contract diagnostics.
func parseContract(t *testing.T, src string) *contract.Contract {
	t.Helper()
	c, diags := contract.Parse([]byte(src), "test")
	if c == nil {
		t.Fatalf("contract failed to parse: %+v", diags)
	}
	return c
}

func TestExampleValueRules(t *testing.T) {
	c := parseContract(t, `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "HAS_DEFAULT":  {"type": "string", "default": "hello"},
    "SECRET":       {"type": "string", "x-envdoctor-secret": true},
    "ENUM_ONLY":    {"type": "string", "enum": ["debug", "info"]},
    "PATTERN_ONLY": {"type": "string", "pattern": "^[a-z]+$"},
    "TYPE_ONLY":    {"type": "string"},
    "PORT":         {"type": "integer", "default": 3000},
    "RATIO":        {"type": "number", "default": 0.5},
    "DEBUG":        {"type": "boolean", "default": true},
    "OFF":          {"type": "boolean", "default": false},
    "EMPTY_DEFAULT": {"type": "string", "default": ""}
  }
}`)

	got, err := Example(c)
	if err != nil {
		t.Fatalf("Example: %v", err)
	}

	want := "" +
		"DEBUG=true\n" +
		"EMPTY_DEFAULT=\n" +
		"ENUM_ONLY=\n" +
		"HAS_DEFAULT=hello\n" +
		"OFF=false\n" +
		"PATTERN_ONLY=\n" +
		"PORT=3000\n" +
		"RATIO=0.5\n" +
		"SECRET=\n" +
		"TYPE_ONLY=\n"
	if string(got) != want {
		t.Errorf("content mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestExampleDefaultIsCheckedAgainstConstraints(t *testing.T) {
	// Parsing already rejects defaults that violate constraints; verify the
	// generated values use only valid defaults.
	c := parseContract(t, `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "C": {"type": "string", "default": "prod", "enum": ["dev", "prod"]},
    "N": {"type": "integer", "default": 10, "minimum": 0, "maximum": 20}
  }
}`)

	got, err := Example(c)
	if err != nil {
		t.Fatalf("Example: %v", err)
	}
	if string(got) != "C=prod\nN=10\n" {
		t.Errorf("content mismatch:\n%s", got)
	}
}

func TestExampleDescriptionsBecomeComments(t *testing.T) {
	c := parseContract(t, `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "PORT": {"type": "integer", "description": "TCP port to listen on"}
  }
}`)

	got, err := Example(c)
	if err != nil {
		t.Fatalf("Example: %v", err)
	}
	if string(got) != "# TCP port to listen on\nPORT=\n" {
		t.Errorf("content mismatch:\n%s", got)
	}
}

func TestExampleMultilineDescription(t *testing.T) {
	c := parseContract(t, `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "KEY": {"type": "string", "description": "first line\nsecond line\r\nthird line"}
  }
}`)

	got, err := Example(c)
	if err != nil {
		t.Fatalf("Example: %v", err)
	}
	want := "# first line\n# second line\n# third line\nKEY=\n"
	if string(got) != want {
		t.Errorf("content mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

// TestExampleStringDefaultsRoundTrip generates every string default form and
// proves it re-parses to the identical value under the EnvDoctor dotenv
// grammar.
func TestExampleStringDefaultsRoundTrip(t *testing.T) {
	values := map[string]string{
		"VALID_URI":        "postgres://user:password@host:5432/db",
		"SIMPLE_SPACED":    "hello world",
		"LEADING_SPACE":    " value",
		"TRAILING_SPACE":   "value ",
		"HASH_MIDDLE":      "a#b",
		"DOLLAR_LITERAL":   "100$5",
		"TAB":              "a\tb",
		"BACKSLASH":        `a\b`,
		"EQUALS":           "a=b=c",
		"STARTS_QUOTE":     "single' and \" double",
		"INTERPOLATION":    "$HOME/fixtures",
		"BRACES":           "${VAR}",
		"SUBSTITUTION":     "$(cmd)",
		"SINGLE_QUOTED":    "'tick'",
		"DOUBLE_QUOTED":    "\"quoted\"",
		"NEWLINE":          "line one\nline two",
		"CRLF_VALUE":       "a\r\nb",
		"BACKSLASH_N":      `a\nb`,
		"INTERP_TAB":       "$X\tY",
		"INTERP_BACKSLASH": "$X\\Y",
	}

	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		want := values[name]
		t.Run(name, func(t *testing.T) {
			def, err := json.Marshal(want)
			if err != nil {
				t.Fatalf("marshal default: %v", err)
			}
			src := fmt.Sprintf(`{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "KEY": {"type": "string", "default": %s}
  }
}`, def)
			c := parseContract(t, src)

			got, err := Example(c)
			if err != nil {
				t.Fatalf("Example: %v", err)
			}
			env, err := source.ParseDotenv(got)
			if err != nil {
				t.Fatalf("generated file does not re-parse: %v\n%s", err, got)
			}
			if env["KEY"] != want {
				t.Errorf("round trip failed\nwant: %q\ngot:  %q\ntext:\n%s", want, env["KEY"], got)
			}
		})
	}
}

func TestExampleNilContract(t *testing.T) {
	if _, err := Example(nil); err == nil {
		t.Fatal("Example(nil) succeeded, want error")
	}
}

// TestExampleUsesLFLineEndings locks the cross-platform invariant that
// generated .env.example documents use LF line endings on every host operating
// system. Multiline descriptions and CR-containing defaults must still render
// without a raw carriage-return byte.
func TestExampleUsesLFLineEndings(t *testing.T) {
	c := parseContract(t, `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "DESC": {"type": "integer", "description": "two\nlines"},
    "CRV":  {"type": "string", "default": "a\rb"}
  }
}`)

	got, err := Example(c)
	if err != nil {
		t.Fatalf("Example: %v", err)
	}
	if bytes.ContainsRune(got, '\r') {
		t.Errorf("generated example must use LF line endings, found CR:\n%q", got)
	}
}

func TestExampleInvalidVariableName(t *testing.T) {
	c := parseContract(t, `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "BAD KEY": {"type": "string", "default": "x"}
  }
}`)

	if _, err := Example(c); err == nil {
		t.Fatal("Example succeeded with an inexpressible variable name, want error")
	}
}

// TestExampleSecretGuard constructs a contract by hand where a secret carries a
// default (the parser rejects this combination; the test proves the generator
// still refuses to emit it).
func TestExampleSecretGuard(t *testing.T) {
	c := &contract.Contract{
		Variables: []*contract.Variable{
			{Name: "API_KEY", Type: contract.TypeString, Secret: true, Default: "super-secret-password"},
			{Name: "NORMAL", Type: contract.TypeString, Default: "x"},
		},
	}
	got, err := Example(c)
	if err != nil {
		t.Fatalf("Example: %v", err)
	}
	if strings.Contains(string(got), "super-secret-password") {
		t.Fatalf("secret default leaked into generated file:\n%s", got)
	}
	if string(got) != "API_KEY=\nNORMAL=x\n" {
		t.Errorf("content mismatch:\n%s", got)
	}
}

func TestExampleDeterministic(t *testing.T) {
	c := parseContract(t, `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "B": {"type": "string", "default": "two"},
    "A": {"type": "integer", "default": 1},
    "C": {"type": "string", "default": "three\nlines\nof text"}
  }
}`)

	first, err := Example(c)
	if err != nil {
		t.Fatalf("Example: %v", err)
	}
	for i := 0; i < 5; i++ {
		next, err := Example(c)
		if err != nil {
			t.Fatalf("Example: %v", err)
		}
		if !bytes.Equal(first, next) {
			t.Fatalf("generated content is not deterministic\nfirst:\n%s\nnext:\n%s", first, next)
		}
	}
}

func TestWriteExampleCreates(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env.example")
	if err := WriteExample(path, []byte("A=1\n")); err != nil {
		t.Fatalf("WriteExample: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(content) != "A=1\n" {
		t.Errorf("content = %q, want %q", content, "A=1\n")
	}
}

func TestWriteExampleRefusesOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env.example")
	if err := os.WriteFile(path, []byte("KEEP=ME\n"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := WriteExample(path, []byte("NEW=1\n")); err == nil {
		t.Fatal("WriteExample overwrote an existing file, want error")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(content) != "KEEP=ME\n" {
		t.Errorf("existing file was modified: %q", content)
	}
}
