package validate

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/haneefojay/envdoctor/internal/contract"
	"github.com/haneefojay/envdoctor/internal/diagnostic"
	"github.com/haneefojay/envdoctor/internal/source"
)

func testContract(vars ...*contract.Variable) *contract.Contract {
	c := &contract.Contract{}
	for _, v := range vars {
		c.Variables = append(c.Variables, v)
	}
	// Match the contract package invariant: variables are sorted by name.
	sort.Slice(c.Variables, func(i, j int) bool { return c.Variables[i].Name < c.Variables[j].Name })
	return c
}

func v(name string, typ contract.Type) *contract.Variable {
	return &contract.Variable{Name: name, Type: typ}
}

func ptrI(n int) *int              { return &n }
func ptrF(f float64) *float64      { return &f }
func strList(s ...string) []string { return s }

func TestValidEnvironment(t *testing.T) {
	c := testContract(
		&contract.Variable{Name: "HOST", Type: contract.TypeString, Required: true},
		&contract.Variable{Name: "PORT", Type: contract.TypeInteger},
		&contract.Variable{Name: "DEBUG", Type: contract.TypeBoolean},
	)
	env := source.Environment{"HOST": "localhost", "PORT": "3000", "DEBUG": "true"}
	if diags := Validate(c, env); len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %+v", diags)
	}
}

func TestRequiredMissing(t *testing.T) {
	c := testContract(
		&contract.Variable{Name: "A", Type: contract.TypeString, Required: true},
		&contract.Variable{Name: "B", Type: contract.TypeString},
	)
	diags := Validate(c, source.Environment{"B": "x"})
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %+v", diags)
	}
	check(t, diags[0], diagnostic.SeverityError, diagnostic.CodeEnvMissing, "A")
}

func TestRequiredEmpty(t *testing.T) {
	c := testContract(&contract.Variable{Name: "A", Type: contract.TypeString, Required: true})
	diags := Validate(c, source.Environment{"A": ""})
	if len(diags) != 1 {
		t.Fatalf("expected 1 ENV_EMPTY diagnostic, got %+v", diags)
	}
	check(t, diags[0], diagnostic.SeverityError, diagnostic.CodeEnvEmpty, "A")
}

func TestRequiredEmptyDoesNotDoubleReport(t *testing.T) {
	// An empty required integer must report ENV_EMPTY, not also ENV_TYPE_MISMATCH.
	c := testContract(&contract.Variable{Name: "A", Type: contract.TypeInteger, Required: true})
	diags := Validate(c, source.Environment{"A": ""})
	if len(diags) != 1 {
		t.Fatalf("expected exactly 1 diagnostic, got %+v", diags)
	}
	check(t, diags[0], diagnostic.SeverityError, diagnostic.CodeEnvEmpty, "A")
}

func TestOptionalEmptyIsValidated(t *testing.T) {
	// An optional variable present with an empty value is validated normally:
	// empty is a valid string but an invalid integer.
	c := testContract(
		&contract.Variable{Name: "S", Type: contract.TypeString},
		&contract.Variable{Name: "I", Type: contract.TypeInteger},
	)
	diags := Validate(c, source.Environment{"S": "", "I": ""})
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %+v", diags)
	}
	check(t, diags[0], diagnostic.SeverityError, diagnostic.CodeEnvTypeMismatch, "I")
}

func TestTypeMismatch(t *testing.T) {
	cases := []struct {
		typ contract.Type
		raw string
	}{
		{contract.TypeBoolean, "yes"},
		{contract.TypeBoolean, "1"},
		{contract.TypeInteger, "3.14"},
		{contract.TypeInteger, "1e3"},
		{contract.TypeNumber, "1e3"},
		{contract.TypeNumber, "NaN"},
	}
	for _, tc := range cases {
		c := testContract(&contract.Variable{Name: "X", Type: tc.typ, Required: true})
		diags := Validate(c, source.Environment{"X": tc.raw})
		if len(diags) != 1 {
			t.Fatalf("%s=%q: expected 1 diagnostic, got %+v", tc.typ, tc.raw, diags)
		}
		check(t, diags[0], diagnostic.SeverityError, diagnostic.CodeEnvTypeMismatch, "X")
	}
}

func TestEnumAndConst(t *testing.T) {
	enum := &contract.Variable{Name: "ENV",
		Type: contract.TypeString, Required: true,
		Enum: []contract.Value{"dev", "prod"},
	}
	c := testContract(enum)
	diags := Validate(c, source.Environment{"ENV": "DEV"})
	check(t, diags[0], diagnostic.SeverityError, diagnostic.CodeEnvInvalidEnum, "ENV")

	diags = Validate(c, source.Environment{"ENV": "dev"})
	if len(diags) != 0 {
		t.Fatalf("expected valid, got %+v", diags)
	}

	cnst := &contract.Variable{Name: "MODE", Type: contract.TypeString, Const: "strict"}
	c = testContract(cnst)
	diags = Validate(c, source.Environment{"MODE": "loose"})
	check(t, diags[0], diagnostic.SeverityError, diagnostic.CodeEnvInvalidEnum, "MODE")
	diags = Validate(c, source.Environment{"MODE": "strict"})
	if len(diags) != 0 {
		t.Fatalf("expected valid, got %+v", diags)
	}
}

func TestPatternMismatch(t *testing.T) {
	c := testContract(&contract.Variable{
		Name: "ID", Type: contract.TypeString, Required: true,
		PatternRaw: "^[a-z]+$",
		Pattern:    regexpMust("^[a-z]+$"),
	})
	diags := Validate(c, source.Environment{"ID": "ABC123"})
	check(t, diags[0], diagnostic.SeverityError, diagnostic.CodeEnvPatternMismatch, "ID")
	if !strings.Contains(diags[0].Message, "^[a-z]+$") {
		t.Errorf("message should reference the pattern, got %q", diags[0].Message)
	}
	diags = Validate(c, source.Environment{"ID": "abc"})
	if len(diags) != 0 {
		t.Fatalf("expected valid, got %+v", diags)
	}
}

func TestLengthOutOfRange(t *testing.T) {
	c := testContract(&contract.Variable{
		Name: "CODE", Type: contract.TypeString, Required: true,
		MinLength: ptrI(2), MaxLength: ptrI(4),
	})
	for _, raw := range []string{"a", "abcde"} {
		diags := Validate(c, source.Environment{"CODE": raw})
		check(t, diags[0], diagnostic.SeverityError, diagnostic.CodeEnvLengthOutOfRange, "CODE")
	}
	// Unicode length is measured in runes.
	c = testContract(&contract.Variable{Name: "CODE", Type: contract.TypeString, MinLength: ptrI(2)})
	if diags := Validate(c, source.Environment{"CODE": "日本"}); len(diags) != 0 {
		t.Fatalf("rune length should satisfy constraint, got %+v", diags)
	}
}

func TestNumberOutOfRange(t *testing.T) {
	c := testContract(&contract.Variable{
		Name: "PORT", Type: contract.TypeInteger, Required: true,
		Minimum: ptrF(1000), Maximum: ptrF(9999),
	})
	for _, raw := range []string{"80", "10000"} {
		diags := Validate(c, source.Environment{"PORT": raw})
		check(t, diags[0], diagnostic.SeverityError, diagnostic.CodeEnvNumberOutOfRange, "PORT")
	}
	// Bounds are inclusive.
	if diags := Validate(c, source.Environment{"PORT": "1000"}); len(diags) != 0 {
		t.Fatalf("minimum is inclusive, got %+v", diags)
	}

	c = testContract(&contract.Variable{
		Name: "RATE", Type: contract.TypeNumber, Required: true,
		ExclusiveMinimum: ptrF(0), ExclusiveMaximum: ptrF(1),
	})
	for _, raw := range []string{"0", "1"} {
		diags := Validate(c, source.Environment{"RATE": raw})
		check(t, diags[0], diagnostic.SeverityError, diagnostic.CodeEnvNumberOutOfRange, "RATE")
	}
}

func TestMultipleOf(t *testing.T) {
	c := testContract(&contract.Variable{
		Name: "CHUNK", Type: contract.TypeNumber, Required: true,
		MultipleOf: ptrF(0.25),
	})
	if diags := Validate(c, source.Environment{"CHUNK": "1.5"}); len(diags) != 0 {
		t.Fatalf("1.5 is a multiple of 0.25, got %+v", diags)
	}
	diags := Validate(c, source.Environment{"CHUNK": "1.3"})
	check(t, diags[0], diagnostic.SeverityError, diagnostic.CodeEnvMultipleOf, "CHUNK")
}

func TestFormatEmailAndURI(t *testing.T) {
	c := testContract(
		&contract.Variable{Name: "EMAIL", Type: contract.TypeString, Format: contract.FormatEmail},
		&contract.Variable{Name: "URL", Type: contract.TypeString, Format: contract.FormatURI},
	)
	if diags := Validate(c, source.Environment{"EMAIL": "dev@example.com", "URL": "https://example.com/path"}); len(diags) != 0 {
		t.Fatalf("expected valid, got %+v", diags)
	}
	diags := Validate(c, source.Environment{"EMAIL": "not-an-email", "URL": "not a url"})
	if len(diags) != 2 {
		t.Fatalf("expected 2 diagnostics, got %+v", diags)
	}
	check(t, diags[0], diagnostic.SeverityError, diagnostic.CodeEnvInvalidFormat, "EMAIL")
	check(t, diags[1], diagnostic.SeverityError, diagnostic.CodeEnvInvalidFormat, "URL")
}

func TestUnknownVariables(t *testing.T) {
	c := testContract(&contract.Variable{Name: "KNOWN", Type: contract.TypeString})
	diags := Validate(c, source.Environment{"KNOWN": "a", "PATH": "x", "CI": "true"})
	if len(diags) != 2 {
		t.Fatalf("expected 2 warnings, got %+v", diags)
	}
	if diags[0].Code != diagnostic.CodeEnvUnknown || diags[1].Code != diagnostic.CodeEnvUnknown {
		t.Fatalf("expected ENV_UNKNOWN diagnostics, got %+v", diags)
	}
	if diags[0].Variable != "CI" || diags[1].Variable != "PATH" {
		t.Fatalf("unknown variables must be ordered deterministically, got %q and %q", diags[0].Variable, diags[1].Variable)
	}
	if diags[0].Severity != diagnostic.SeverityWarning {
		t.Fatalf("unknown variables must be warnings, got %v", diags[0].Severity)
	}
}

func TestDeterministicOrdering(t *testing.T) {
	c := testContract(
		&contract.Variable{Name: "ZED", Type: contract.TypeInteger, Required: true},
		&contract.Variable{Name: "ALPHA", Type: contract.TypeInteger, Required: true},
		&contract.Variable{Name: "MID", Type: contract.TypeBoolean, Required: true},
	)
	diags := Validate(c, source.Environment{"ZED": "bad", "ALPHA": "bad", "MID": "true"})
	if len(diags) != 2 {
		t.Fatalf("expected 2 diagnostics, got %+v", diags)
	}
	if diags[0].Variable != "ALPHA" || diags[1].Variable != "ZED" {
		t.Fatalf("diagnostics must follow variable-name order, got %q then %q", diags[0].Variable, diags[1].Variable)
	}
}

func TestSecretValuesNeverLeak(t *testing.T) {
	const secret = "s3cr3t-t0k3n-9zYxWv"
	secretInt := "987654321"
	secretBool := "true"

	c := testContract(
		&contract.Variable{Name: "TOKEN", Type: contract.TypeString, Required: true, Secret: true, MinLength: ptrI(40)},
		&contract.Variable{Name: "PIN", Type: contract.TypeInteger, Required: true, Secret: true, Maximum: ptrF(100)},
		&contract.Variable{Name: "FLAG", Type: contract.TypeBoolean, Required: true, Secret: true},
	)
	diags := Validate(c, source.Environment{"TOKEN": secret, "PIN": secretInt, "FLAG": secretBool})
	if len(diags) != 2 {
		t.Fatalf("expected the two failing secret diagnostics, got %+v", diags)
	}
	for _, d := range diags {
		for _, leaked := range []string{secret, secretInt, secretBool} {
			if strings.Contains(d.Message, leaked) {
				t.Fatalf("secret value leaked in diagnostic: %q", d.Message)
			}
		}
	}
}

func TestNilContract(t *testing.T) {
	diags := Validate(nil, source.Environment{})
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %+v", diags)
	}
	check(t, diags[0], diagnostic.SeverityError, diagnostic.CodeContractInvalid, "")
}

func TestCaseSensitivityOfNames(t *testing.T) {
	// Variable names are semantically case-sensitive even on case-insensitive
	// hosts.
	c := testContract(&contract.Variable{Name: "API_KEY", Type: contract.TypeString, Required: true})
	diags := Validate(c, source.Environment{"api_key": "x"})
	// api_key is not API_KEY: the declared variable is missing (and the
	// lowercase entry surfaces separately as an unknown-variable warning). The
	// variable name is carried in Variable, not embedded in the message.
	check(t, diags[0], diagnostic.SeverityError, diagnostic.CodeEnvMissing, "API_KEY")
}

func check(t *testing.T, d diagnostic.Diagnostic, sev diagnostic.Severity, code diagnostic.Code, variable string) {
	t.Helper()
	if d.Severity != sev {
		t.Errorf("severity = %v, want %v", d.Severity, sev)
	}
	if d.Code != code {
		t.Errorf("code = %v, want %v", d.Code, code)
	}
	if d.Variable != variable {
		t.Errorf("variable = %q, want %q", d.Variable, variable)
	}
}

func regexpMust(p string) *regexp.Regexp { return regexp.MustCompile(p) }
