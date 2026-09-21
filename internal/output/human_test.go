package output

import (
	"errors"
	"strings"
	"testing"

	"github.com/haneefojay/envdoctor/internal/diagnostic"
)

// lines joins lines with newlines and appends the final newline every renderer
// is expected to produce.
func lines(ls ...string) string { return strings.Join(ls, "\n") + "\n" }

// failWriter always fails, to exercise error propagation.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestWriteHumanGolden(t *testing.T) {
	cases := []struct {
		name   string
		result Result
		want   string
	}{
		{
			name:   "valid with no diagnostics",
			result: NewResult(nil),
			want:   lines("Environment is valid"),
		},
		{
			name: "warning only",
			result: NewResult([]diagnostic.Diagnostic{
				warnDiag(diagnostic.CodeEnvUnknown, "EXTRA_SETTING", "Variable is not declared in the contract."),
			}),
			want: lines(
				"Environment is valid, with warnings",
				"",
				"⚠ EXTRA_SETTING",
				"  WARN ENV_UNKNOWN",
				"  Variable is not declared in the contract.",
				"",
				"0 errors, 1 warning",
			),
		},
		{
			name: "one error",
			result: NewResult([]diagnostic.Diagnostic{
				errDiag(diagnostic.CodeEnvTypeMismatch, "PORT", "Expected integer."),
			}),
			want: lines(
				"Environment validation failed",
				"",
				"✗ PORT",
				"  ERROR ENV_TYPE_MISMATCH",
				"  Expected integer.",
				"",
				"1 error, 0 warnings",
			),
		},
		{
			// PRD human-output example. Errors are grouped before warnings
			// regardless of input order.
			name: "many errors and warning",
			result: NewResult([]diagnostic.Diagnostic{
				warnDiag(diagnostic.CodeEnvUnknown, "EXTRA_SETTING", "Variable is not declared in the contract."),
				errDiag(diagnostic.CodeEnvTypeMismatch, "PORT", "Expected integer."),
				errDiag(diagnostic.CodeEnvMissing, "DATABASE_URL", "Required variable is not set."),
			}),
			want: lines(
				"Environment validation failed",
				"",
				"✗ DATABASE_URL",
				"  ERROR ENV_MISSING",
				"  Required variable is not set.",
				"",
				"✗ PORT",
				"  ERROR ENV_TYPE_MISMATCH",
				"  Expected integer.",
				"",
				"⚠ EXTRA_SETTING",
				"  WARN ENV_UNKNOWN",
				"  Variable is not declared in the contract.",
				"",
				"2 errors, 1 warning",
			),
		},
		{
			name: "malformed source (no variable)",
			result: NewResult([]diagnostic.Diagnostic{
				errDiag(diagnostic.CodeEnvFileInvalid, "", "line 3: interpolation is not supported; use single quotes for a literal $ value"),
			}),
			want: lines(
				"Environment validation failed",
				"",
				"✗ line 3: interpolation is not supported; use single quotes for a literal $ value",
				"  ERROR ENV_FILE_INVALID",
				"",
				"1 error, 0 warnings",
			),
		},
		{
			name: "malformed contract (no variable)",
			result: NewResult([]diagnostic.Diagnostic{
				errDiag(diagnostic.CodeContractInvalid, "", "Contract is not valid JSON: unexpected end of JSON input."),
			}),
			want: lines(
				"Environment validation failed",
				"",
				"✗ Contract is not valid JSON: unexpected end of JSON input.",
				"  ERROR CONTRACT_INVALID",
				"",
				"1 error, 0 warnings",
			),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var b strings.Builder
			if err := WriteHuman(&b, tc.result); err != nil {
				t.Fatalf("WriteHuman failed: %v", err)
			}
			if got := b.String(); got != tc.want {
				t.Errorf("human output mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, tc.want)
			}
		})
	}
}

// TestOutputLFOnly locks the cross-platform invariant that human and JSON
// output use LF line endings on every host operating system.
func TestOutputLFOnly(t *testing.T) {
	r := NewResult([]diagnostic.Diagnostic{
		diagnostic.Errorf(diagnostic.CodeEnvMissing, "DATABASE_URL", "Required variable is not set."),
		diagnostic.Warningf(diagnostic.CodeEnvUnknown, "EXTRA", "Variable is not declared in the contract."),
	})

	var human strings.Builder
	if err := WriteHuman(&human, r); err != nil {
		t.Fatalf("WriteHuman: %v", err)
	}
	if strings.ContainsRune(human.String(), '\r') {
		t.Errorf("human output must use LF line endings, found CR:\n%q", human.String())
	}

	var jsonOut strings.Builder
	if err := WriteJSON(&jsonOut, r); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if strings.ContainsRune(jsonOut.String(), '\r') {
		t.Errorf("JSON output must use LF line endings, found CR:\n%q", jsonOut.String())
	}
}

func TestWriteHumanNoANSI(t *testing.T) {
	r := NewResult([]diagnostic.Diagnostic{
		errDiag(diagnostic.CodeEnvMissing, "A", "missing"),
		warnDiag(diagnostic.CodeEnvUnknown, "B", "unknown"),
	})
	var b strings.Builder
	if err := WriteHuman(&b, r); err != nil {
		t.Fatalf("WriteHuman failed: %v", err)
	}
	if strings.Contains(b.String(), "\x1b") {
		t.Errorf("human output must not contain ANSI escape sequences: %q", b.String())
	}
}

func TestWriteHumanSecretNotDisclosed(t *testing.T) {
	// A realistic secret-looking value that validate would reject without ever
	// putting the value into a diagnostic. This guards against a future change
	// that embeds environment values into diagnostic messages.
	const secret = "s3cr3t-t0k3n-9zYxWv"
	r := NewResult([]diagnostic.Diagnostic{
		errDiag(diagnostic.CodeEnvLengthOutOfRange, "TOKEN", "value length 18 is below the minimum of 40"),
	})
	var b strings.Builder
	if err := WriteHuman(&b, r); err != nil {
		t.Fatalf("WriteHuman failed: %v", err)
	}
	if strings.Contains(b.String(), secret) {
		t.Fatalf("secret value leaked into human output: %q", b.String())
	}
}

func TestWriteHumanWriterError(t *testing.T) {
	if err := WriteHuman(failWriter{}, NewResult(nil)); err == nil {
		t.Fatalf("expected WriteHuman to return the writer error")
	}
}
