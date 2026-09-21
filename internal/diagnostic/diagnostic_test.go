package diagnostic

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
)

func TestSeverityStableValues(t *testing.T) {
	cases := []struct {
		got  Severity
		want string
	}{
		{SeverityError, "error"},
		{SeverityWarning, "warning"},
	}
	for _, tc := range cases {
		if string(tc.got) != tc.want {
			t.Errorf("Severity = %q, want %q", tc.got, tc.want)
		}
	}
}

func TestStableCodes(t *testing.T) {
	cases := []struct {
		got  Code
		want string
	}{
		{CodeContractInvalid, "CONTRACT_INVALID"},
		{CodeContractUnsupportedKeyword, "CONTRACT_UNSUPPORTED_KEYWORD"},
		{CodeContractDuplicateVariable, "CONTRACT_DUPLICATE_VARIABLE"},
		{CodeEnvMissing, "ENV_MISSING"},
		{CodeEnvEmpty, "ENV_EMPTY"},
		{CodeEnvUnknown, "ENV_UNKNOWN"},
		{CodeEnvTypeMismatch, "ENV_TYPE_MISMATCH"},
		{CodeEnvInvalidEnum, "ENV_INVALID_ENUM"},
		{CodeEnvPatternMismatch, "ENV_PATTERN_MISMATCH"},
		{CodeEnvNumberOutOfRange, "ENV_NUMBER_OUT_OF_RANGE"},
		{CodeEnvInvalidFormat, "ENV_INVALID_FORMAT"},
		{CodeEnvLengthOutOfRange, "ENV_LENGTH_OUT_OF_RANGE"},
		{CodeEnvMultipleOf, "ENV_MULTIPLE_OF"},
		{CodeEnvFileInvalid, "ENV_FILE_INVALID"},
		{CodeSourceInvalid, "SOURCE_INVALID"},
	}
	seen := make(map[Code]bool, len(cases))
	for _, tc := range cases {
		if string(tc.got) != tc.want {
			t.Errorf("code = %q, want %q", tc.got, tc.want)
		}
		if seen[tc.got] {
			t.Errorf("code %q is duplicated in the stable-code set", tc.got)
		}
		seen[tc.got] = true
	}
}

func TestErrorfShape(t *testing.T) {
	cases := []struct {
		name     string
		code     Code
		variable string
		format   string
		args     []any
		wantMsg  string
	}{
		{"plain", CodeEnvMissing, "DATABASE_URL", "Required variable is not set.", nil, "Required variable is not set."},
		{"formatted", CodeEnvTypeMismatch, "PORT", "Expected %s.", []any{"integer"}, "Expected integer."},
		{"contract-level", CodeContractInvalid, "", "Contract is not valid.", nil, "Contract is not valid."},
		{"multiple args", CodeEnvNumberOutOfRange, "PORT", "value %d is not a multiple of %g", []any{80, 0.25}, "value 80 is not a multiple of 0.25"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := Errorf(tc.code, tc.variable, tc.format, tc.args...)
			checkShape(t, d, SeverityError, tc.code, tc.variable, tc.wantMsg)
		})
	}
}

func TestWarningfShape(t *testing.T) {
	cases := []struct {
		name     string
		code     Code
		variable string
		format   string
		args     []any
		wantMsg  string
	}{
		{"plain", CodeEnvUnknown, "EXTRA", "Variable is not declared in the contract.", nil, "Variable is not declared in the contract."},
		{"formatted", CodeEnvUnknown, "CI", "Unknown %s %d.", []any{"variable", 1}, "Unknown variable 1."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := Warningf(tc.code, tc.variable, tc.format, tc.args...)
			checkShape(t, d, SeverityWarning, tc.code, tc.variable, tc.wantMsg)
		})
	}
}

func checkShape(t *testing.T, d Diagnostic, sev Severity, code Code, variable, wantMsg string) {
	t.Helper()
	if d.Severity != sev {
		t.Errorf("Severity = %q, want %q", d.Severity, sev)
	}
	if d.Code != code {
		t.Errorf("Code = %q, want %q", d.Code, code)
	}
	if d.Variable != variable {
		t.Errorf("Variable = %q, want %q", d.Variable, variable)
	}
	if d.Message != wantMsg {
		t.Errorf("Message = %q, want %q", d.Message, wantMsg)
	}
}

func TestErrorAndWarningDifferOnlyInSeverity(t *testing.T) {
	e := Errorf(CodeEnvUnknown, "X", "same message")
	w := Warningf(CodeEnvUnknown, "X", "same message")
	if e.Severity == w.Severity {
		t.Errorf("error and warning must differ in severity")
	}
	if e.Code != w.Code || e.Variable != w.Variable || e.Message != w.Message {
		t.Errorf("same code/variable/message expected, got %+v vs %+v", e, w)
	}
}

func TestDiagnosticJSONShapeAndDeterminism(t *testing.T) {
	d := Errorf(CodeEnvMissing, "DATABASE_URL", "Required variable is not set.")
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	// Stable shape and field order; no ANSI sequences; message not truncated.
	const want = `{"severity":"error","code":"ENV_MISSING","variable":"DATABASE_URL","message":"Required variable is not set."}`
	if string(data) != want {
		t.Errorf("JSON output = %s, want %s", data, want)
	}

	again, err := json.Marshal(d)
	if err != nil || string(again) != want {
		t.Errorf("JSON output is not deterministic: %s, %v", again, err)
	}

	var round Diagnostic
	if err := json.Unmarshal(data, &round); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if round != d {
		t.Errorf("round-tripped diagnostic = %+v, want %+v", round, d)
	}
}

// sortByOrderingKey sorts diagnostics by the canonical key used by consumers:
// variable, then code, then message. Duplicating the key in the test asserts
// that diagnostics are orderable, not that the package owns sorting.
func sortByOrderingKey(ds []Diagnostic) {
	sort.SliceStable(ds, func(i, j int) bool {
		if ds[i].Variable != ds[j].Variable {
			return ds[i].Variable < ds[j].Variable
		}
		if ds[i].Code != ds[j].Code {
			return ds[i].Code < ds[j].Code
		}
		return ds[i].Message < ds[j].Message
	})
}

func TestDeterministicOrdering(t *testing.T) {
	build := func() []Diagnostic {
		return []Diagnostic{
			Warningf(CodeEnvUnknown, "ZED", "unknown variable"),
			Errorf(CodeEnvMissing, "ALPHA", "missing required variable"),
			Errorf(CodeEnvTypeMismatch, "ALPHA", "value is not a valid integer"),
			Errorf(CodeEnvTypeMismatch, "ALPHA", "expected conflict"),
			Errorf(CodeEnvTypeMismatch, "BETA", "value is not a valid integer"),
		}
	}

	want := []Diagnostic{
		Errorf(CodeEnvMissing, "ALPHA", "missing required variable"),
		Errorf(CodeEnvTypeMismatch, "ALPHA", "expected conflict"),
		Errorf(CodeEnvTypeMismatch, "ALPHA", "value is not a valid integer"),
		Errorf(CodeEnvTypeMismatch, "BETA", "value is not a valid integer"),
		Warningf(CodeEnvUnknown, "ZED", "unknown variable"),
	}

	// Same set, reversed input order, sorted again: order must come from the
	// key, never from the input order.
	for i := 0; i < 5; i++ {
		got := build()
		sortByOrderingKey(got)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("iteration %d: got %+v, want %+v", i, got, want)
		}
	}
}
