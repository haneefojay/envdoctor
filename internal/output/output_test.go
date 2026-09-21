package output

import (
	"reflect"
	"testing"

	"github.com/haneefojay/envdoctor/internal/diagnostic"
)

// diag builds a diagnostic with an exact message (no formatting).
func diag(sev diagnostic.Severity, code diagnostic.Code, variable, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{Severity: sev, Code: code, Variable: variable, Message: message}
}

func errDiag(code diagnostic.Code, variable, message string) diagnostic.Diagnostic {
	return diag(diagnostic.SeverityError, code, variable, message)
}

func warnDiag(code diagnostic.Code, variable, message string) diagnostic.Diagnostic {
	return diag(diagnostic.SeverityWarning, code, variable, message)
}

func TestValidSemantics(t *testing.T) {
	cases := []struct {
		name   string
		result Result
		want   bool
	}{
		{"empty result", NewResult(nil), true},
		{"warning only", NewResult([]diagnostic.Diagnostic{
			warnDiag(diagnostic.CodeEnvUnknown, "EXTRA", "unknown"),
		}), true},
		{"error", NewResult([]diagnostic.Diagnostic{
			errDiag(diagnostic.CodeEnvMissing, "A", "missing"),
		}), false},
		{"mixed", NewResult([]diagnostic.Diagnostic{
			warnDiag(diagnostic.CodeEnvUnknown, "EXTRA", "unknown"),
			errDiag(diagnostic.CodeEnvMissing, "A", "missing"),
		}), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.result.Valid(); got != tc.want {
				t.Errorf("Valid() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNewResultCanonicalOrder(t *testing.T) {
	// Deliberately scrambled input order.
	input := []diagnostic.Diagnostic{
		warnDiag(diagnostic.CodeEnvUnknown, "ZED", "z"),
		errDiag(diagnostic.CodeEnvTypeMismatch, "BETA", "b"),
		warnDiag(diagnostic.CodeEnvUnknown, "ALPHA", "a"),
		errDiag(diagnostic.CodeEnvTypeMismatch, "ALPHA", "c"),
		errDiag(diagnostic.CodeEnvMissing, "ALPHA", "a"),
	}
	want := []diagnostic.Diagnostic{
		errDiag(diagnostic.CodeEnvMissing, "ALPHA", "a"),
		errDiag(diagnostic.CodeEnvTypeMismatch, "ALPHA", "c"),
		errDiag(diagnostic.CodeEnvTypeMismatch, "BETA", "b"),
		warnDiag(diagnostic.CodeEnvUnknown, "ALPHA", "a"),
		warnDiag(diagnostic.CodeEnvUnknown, "ZED", "z"),
	}

	got := NewResult(input).Diagnostics()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("canonical order mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestNewResultDoesNotMutateInput(t *testing.T) {
	input := []diagnostic.Diagnostic{
		warnDiag(diagnostic.CodeEnvUnknown, "ZED", "z"),
		errDiag(diagnostic.CodeEnvMissing, "ALPHA", "a"),
	}
	original := make([]diagnostic.Diagnostic, len(input))
	copy(original, input)

	_ = NewResult(input)

	if !reflect.DeepEqual(input, original) {
		t.Errorf("NewResult mutated its input: got %+v, want %+v", input, original)
	}
}

func TestNewResultDeterministic(t *testing.T) {
	build := func() []diagnostic.Diagnostic {
		return []diagnostic.Diagnostic{
			warnDiag(diagnostic.CodeEnvUnknown, "B", "b"),
			errDiag(diagnostic.CodeEnvTypeMismatch, "A", "b"),
			errDiag(diagnostic.CodeEnvTypeMismatch, "A", "a"),
			warnDiag(diagnostic.CodeEnvUnknown, "A", "a"),
		}
	}
	reverse := func(ds []diagnostic.Diagnostic) []diagnostic.Diagnostic {
		out := make([]diagnostic.Diagnostic, len(ds))
		for i, d := range ds {
			out[len(ds)-1-i] = d
		}
		return out
	}

	first := NewResult(build()).Diagnostics()
	for i := 0; i < 5; i++ {
		got := NewResult(reverse(build())).Diagnostics()
		if !reflect.DeepEqual(got, first) {
			t.Fatalf("iteration %d: ordering depends on input order:\n got %+v\nwant %+v", i, got, first)
		}
	}
}

func TestDiagnosticsReturnsCopy(t *testing.T) {
	r := NewResult([]diagnostic.Diagnostic{
		errDiag(diagnostic.CodeEnvMissing, "A", "missing"),
	})
	got := r.Diagnostics()
	got[0] = errDiag(diagnostic.CodeEnvEmpty, "A", "empty")

	if reflect.DeepEqual(r.Diagnostics(), got) {
		t.Errorf("mutating the returned slice must not change the Result")
	}
}

func TestCounts(t *testing.T) {
	r := NewResult([]diagnostic.Diagnostic{
		errDiag(diagnostic.CodeEnvMissing, "A", "missing"),
		errDiag(diagnostic.CodeEnvEmpty, "B", "empty"),
		warnDiag(diagnostic.CodeEnvUnknown, "C", "unknown"),
	})
	errors, warnings := r.counts()
	if errors != 2 || warnings != 1 {
		t.Errorf("counts() = (%d, %d), want (2, 1)", errors, warnings)
	}
}
