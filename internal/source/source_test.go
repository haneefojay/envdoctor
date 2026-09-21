package source

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFromProcessPreservesValues(t *testing.T) {
	entries := []string{
		"PLAIN=value",
		"SPACES=  value with  spaces  ",
		"EMPTY=",
		"CASE=upper",
		"case=lower",
		"EQ=a=b=c",
		"UNICODE=héllo wörld",
	}
	env := FromProcess(entries)
	want := Environment{
		"PLAIN":   "value",
		"SPACES":  "  value with  spaces  ",
		"EMPTY":   "",
		"CASE":    "upper",
		"case":    "lower",
		"EQ":      "a=b=c",
		"UNICODE": "héllo wörld",
	}
	assertEnv(t, env, want)
}

func TestFromProcessDuplicateLastWins(t *testing.T) {
	env := FromProcess([]string{"A=first", "A=second"})
	if env["A"] != "second" {
		t.Errorf("A = %q, want second", env["A"])
	}
}

func TestFromProcessSkipsEntriesWithoutEquals(t *testing.T) {
	env := FromProcess([]string{"MALFORMED", "OK=1"})
	if _, ok := env["MALFORMED"]; ok {
		t.Errorf("entry without = must be skipped")
	}
	if env["OK"] != "1" {
		t.Errorf("OK = %q, want 1", env["OK"])
	}
}

func TestParseDotenvValid(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		want Environment
	}{
		{
			"simple",
			"A=1\nB=two\nC=three=four",
			Environment{"A": "1", "B": "two", "C": "three=four"},
		},
		{
			"blank and comments",
			"\n# full line comment\n   # indented comment\nA=1\n\nB=2\n",
			Environment{"A": "1", "B": "2"},
		},
		{
			"no trailing newline",
			"A=1\nB=2",
			Environment{"A": "1", "B": "2"},
		},
		{
			"empty values present",
			"EMPTY=\nDQ=\nSQ=''\nSP=\n",
			Environment{"EMPTY": "", "DQ": "", "SQ": "", "SP": ""},
		},
		{
			"single quoted literal",
			"NAME='value'\nDOLLAR='$HOME'\n",
			Environment{"NAME": "value", "DOLLAR": "$HOME"},
		},
		{
			"double quoted escapes",
			"PATH=\"C:\\\\tmp\"\nLINE=\"a\\nb\"\nTAB=\"a\\tb\"\nQ=\"say \\\"hi\\\"\"\nCR=\"a\\rb\"\nD=\"\\$HOME\"\nLIT=\"100$\"\n",
			Environment{
				"PATH": "C:\\tmp",
				"LINE": "a\nb",
				"TAB":  "a\tb",
				"Q":    "say \"hi\"",
				"CR":   "a\rb",
				"D":    "$HOME",
				"LIT":  "100$",
			},
		},
		{
			"key surrounding whitespace trimmed",
			"A =1\nB = 2\nC=3 = 4\n",
			// Only the key is trimmed; the value is preserved verbatim.
			Environment{"A": "1", "B": " 2", "C": "3 = 4"},
		},
		{
			"value whitespace preserved",
			"A=  leading\nB=trailing  \nC= a b c \n",
			Environment{"A": "  leading", "B": "trailing  ", "C": " a b c "},
		},
		{
			"crlf",
			"A=1\r\nB=2\r\n\r\n# comment\r\nC=3\r\n",
			Environment{"A": "1", "B": "2", "C": "3"},
		},
		{
			"unicode",
			"GREETING=héllo\nJAPAN=日本\nEMOJI=🚀\n",
			Environment{"GREETING": "héllo", "JAPAN": "日本", "EMOJI": "🚀"},
		},
		{
			"dollar not interpolation",
			"PRICE=100$\nPERCENT=50%$\nBARE=$5\n",
			Environment{"PRICE": "100$", "PERCENT": "50%$", "BARE": "$5"},
		},
		{
			"quotes inside double quotes literal",
			"V=\"it's fine\"\nDQ=\"a\\\"b\"\n",
			Environment{"V": "it's fine", "DQ": "a\"b"},
		},
		{
			"escaped dollar in double quotes literal",
			"D=\"\\$HOME\"\n",
			Environment{"D": "$HOME"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env, err := ParseDotenv([]byte(tc.doc))
			if err != nil {
				t.Fatalf("ParseDotenv failed: %v", err)
			}
			assertEnv(t, env, tc.want)
		})
	}
}

func TestParseDotenvRejects(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		sub  string
	}{
		{"no equals", "A=1\nJUSTACHUNK\n", "KEY=VALUE"},
		{"empty key", "=value\n", "invalid variable name"},
		{"key starts with digit", "1X=2\n", "invalid variable name"},
		{"key with dash", "A-B=2\n", "invalid variable name"},
		{"key with slash", "PATH/TO=2\n", "invalid variable name"},
		{"duplicate key", "A=1\nA=2\n", "appears more than once"},
		{"duplicate with spaced key", "A=1\n A =2\n", "appears more than once"},
		{"unterminated single quote", "V='abc\n", "unterminated single-quoted value"},
		{"single quote unbalanced", "V='a'b\n", "unbalanced single-quoted value"},
		{"trailing after single quote", "V='a' b\n", "unterminated single-quoted value"},
		{"unterminated double quote", "V=\"abc\n", "unterminated double-quoted value"},
		{"trailing after double quote", "V=\"a\"b\n", "unterminated double-quoted value"},
		{"unsupported escape", "V=\"\\x\"\n", "unsupported escape"},
		{"dangling escape", "V=\"abc\\\"\n", "unterminated"},
		{"interpolation unquoted var", "V=$HOME\n", "interpolation is not supported"},
		{"interpolation unquoted method", "V=foo$BAR baz\n", "interpolation is not supported"},
		{"interpolation unquoted brace", "V=${HOME}\n", "interpolation is not supported"},
		{"interpolation unquoted command", "V=$(whoami)\n", "interpolation is not supported"},
		{"interpolation double quoted", "V=\"x${A}\"\n", "interpolation is not supported"},
		{"interpolation double quoted shorthand", "V=\"x$A\"\n", "interpolation is not supported"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseDotenv([]byte(tc.doc))
			if err == nil {
				t.Fatalf("expected error for %q containing %s", tc.doc, tc.sub)
			}
			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("expected *ParseError, got %T: %v", err, err)
			}
			if pe.Line <= 0 {
				t.Fatalf("expected a 1-based line number, got %d", pe.Line)
			}
			if !IsParseError(err) {
				t.Fatalf("IsParseError must report true")
			}
		})
	}
}

func TestParseDotenvErrorLineNumbers(t *testing.T) {
	_, err := ParseDotenv([]byte("A=1\nB=2\nC=$HOME\n"))
	requireLine(t, err, 3)

	_, err = ParseDotenv([]byte("\n\nDUP=1\nDUP=2\n"))
	requireLine(t, err, 4)

	_, err = ParseDotenv([]byte("A=1\nBAD LINE\n"))
	requireLine(t, err, 2)
}

func requireLine(t *testing.T, err error, want int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ParseError, got %T", err)
	}
	if pe.Line != want {
		t.Fatalf("Line = %d, want %d (%v)", pe.Line, want, err)
	}
}

func TestSingleQuotedDollarLiteral(t *testing.T) {
	// The documented way to express a literal dollar sign.
	env, err := ParseDotenv([]byte("SECRET='p@$$w0rd'\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env["SECRET"] != "p@$$w0rd" {
		t.Errorf("SECRET = %q", env["SECRET"])
	}
}

func TestLoadDotenv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("A=1\r\nB=two\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env, err := LoadDotenv(path)
	if err != nil {
		t.Fatalf("LoadDotenv failed: %v", err)
	}
	assertEnv(t, env, Environment{"A": "1", "B": "two"})
}

func TestLoadDotenvReadError(t *testing.T) {
	_, err := LoadDotenv(filepath.Join(t.TempDir(), "missing.env"))
	if err == nil {
		t.Fatalf("expected read error")
	}
	if IsParseError(err) {
		t.Fatalf("read errors must not be parse errors")
	}
}

func assertEnv(t *testing.T, got, want Environment) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Environment = %v, want %v", got, want)
	}
}
