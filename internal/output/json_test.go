package output

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/haneefojay/envdoctor/internal/diagnostic"
)

func TestWriteJSONGolden(t *testing.T) {
	cases := []struct {
		name   string
		result Result
		want   string
	}{
		{
			name:   "valid with no diagnostics",
			result: NewResult(nil),
			want: lines(
				"{",
				`  "valid": true,`,
				`  "diagnostics": []`,
				"}",
			),
		},
		{
			name: "single error",
			result: NewResult([]diagnostic.Diagnostic{
				errDiag(diagnostic.CodeEnvMissing, "DATABASE_URL", "Required variable is not set."),
			}),
			want: lines(
				"{",
				`  "valid": false,`,
				`  "diagnostics": [`,
				"    {",
				`      "severity": "error",`,
				`      "code": "ENV_MISSING",`,
				`      "variable": "DATABASE_URL",`,
				`      "message": "Required variable is not set."`,
				"    }",
				"  ]",
				"}",
			),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var b strings.Builder
			if err := WriteJSON(&b, tc.result); err != nil {
				t.Fatalf("WriteJSON failed: %v", err)
			}
			if got := b.String(); got != tc.want {
				t.Errorf("JSON output mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, tc.want)
			}
		})
	}
}

func TestWriteJSONShapeAndDeterminism(t *testing.T) {
	r := NewResult([]diagnostic.Diagnostic{
		warnDiag(diagnostic.CodeEnvUnknown, "EXTRA", "unknown environment variable"),
		errDiag(diagnostic.CodeEnvTypeMismatch, "PORT", "value is not a valid integer"),
	})

	var first, second strings.Builder
	if err := WriteJSON(&first, r); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}
	if err := WriteJSON(&second, r); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}
	if first.String() != second.String() {
		t.Fatalf("JSON output is not deterministic:\n%s\n%s", first.String(), second.String())
	}
	if strings.Contains(first.String(), "\x1b") {
		t.Errorf("JSON output must not contain ANSI escape sequences: %q", first.String())
	}

	var doc struct {
		Valid       bool                    `json:"valid"`
		Diagnostics []diagnostic.Diagnostic `json:"diagnostics"`
	}
	if err := json.Unmarshal([]byte(first.String()), &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if doc.Valid {
		t.Errorf("valid = true, want false")
	}
	if len(doc.Diagnostics) != 2 {
		t.Fatalf("diagnostics = %d, want 2", len(doc.Diagnostics))
	}
	// Errors are grouped before warnings, and fields survive the round trip.
	if doc.Diagnostics[0].Variable != "PORT" || doc.Diagnostics[1].Variable != "EXTRA" {
		t.Errorf("unexpected diagnostic order: %+v", doc.Diagnostics)
	}
}

func TestWriteJSONSecretNotDisclosed(t *testing.T) {
	// See the human-renderer equivalent: diagnostics never carry values, and a
	// future change that embeds one must fail this test.
	const secret = "postgres://user:p@ssw0rd@host/db"
	r := NewResult([]diagnostic.Diagnostic{
		errDiag(diagnostic.CodeEnvInvalidFormat, "DATABASE_URL", "value does not have a valid uri format"),
	})
	var b strings.Builder
	if err := WriteJSON(&b, r); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}
	if strings.Contains(b.String(), secret) {
		t.Fatalf("secret value leaked into JSON output: %q", b.String())
	}
}

func TestWriteJSONWriterError(t *testing.T) {
	if err := WriteJSON(failWriter{}, NewResult(nil)); err == nil {
		t.Fatalf("expected WriteJSON to return the writer error")
	}
}
