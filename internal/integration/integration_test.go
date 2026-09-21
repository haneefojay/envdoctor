package integration

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/haneefojay/envdoctor/internal/contract"
	"github.com/haneefojay/envdoctor/internal/diagnostic"
	"github.com/haneefojay/envdoctor/internal/output"
	"github.com/haneefojay/envdoctor/internal/source"
	"github.com/haneefojay/envdoctor/internal/validate"
)

// repoRoot resolves the repository root from this test file's source path,
// independent of the working directory that go test runs in.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root, err := filepath.Abs(filepath.Join(filepath.Dir(file), "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

// contractPath returns the absolute path of a fixture under testdata/contracts.
func contractPath(root, name string) string {
	return filepath.Join(root, "testdata", "contracts", name)
}

// loadContract parses a contract fixture, failing on any contract diagnostic.
func loadContract(t *testing.T, root, name string) *contract.Contract {
	t.Helper()
	data, err := os.ReadFile(contractPath(root, name))
	if err != nil {
		t.Fatalf("read contract fixture: %v", err)
	}
	c, diags := contract.Parse(data, name)
	if c == nil {
		t.Fatalf("contract fixture %s did not parse: %+v", name, diags)
	}
	return c
}

// loadDotenv parses a dotenv fixture under testdata/dotenv.
func loadDotenv(t *testing.T, root, name string) source.Environment {
	t.Helper()
	env, err := source.LoadDotenv(filepath.Join(root, "testdata", "dotenv", name))
	if err != nil {
		t.Fatalf("dotenv fixture %s: %v", name, err)
	}
	return env
}

// loadProcessEnvironment reads a fixture under testdata/environments as raw
// "KEY=VALUE" process-environment entries, then normalizes them the way the
// process source does.
func loadProcessEnvironment(t *testing.T, root, name string) source.Environment {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "testdata", "environments", name))
	if err != nil {
		t.Fatalf("read environment fixture: %v", err)
	}
	var lines []string
	for _, line := range strings.Split(string(raw), "\n") {
		lines = append(lines, strings.TrimSuffix(line, "\r"))
	}
	return source.FromProcess(lines)
}

// sameCodeMultiset reports whether got and want contain the same diagnostics
// codes with the same multiplicities, independent of ordering.
func sameCodeMultiset(got, want []diagnostic.Code) bool {
	if len(got) != len(want) {
		return false
	}
	g := append([]diagnostic.Code(nil), got...)
	w := append([]diagnostic.Code(nil), want...)
	sortCodes(g)
	sortCodes(w)
	for i := range g {
		if g[i] != w[i] {
			return false
		}
	}
	return true
}

func sortCodes(cs []diagnostic.Code) {
	sort.Slice(cs, func(i, j int) bool { return cs[i] < cs[j] })
}

// scenario is one realistic end-to-end pipeline case.
type scenario struct {
	name      string
	contract  string
	dotenv    string // relative to testdata/dotenv
	process   string // relative to testdata/environments
	wantCodes []diagnostic.Code
}

func TestPipelineScenarios(t *testing.T) {
	root := repoRoot(t)
	cases := []scenario{
		{
			name:     "fastapi service valid",
			contract: "fastapi.json",
			dotenv:   "fastapi-valid.env",
		},
		{
			name:     "fastapi service broken values",
			contract: "fastapi.json",
			dotenv:   "fastapi-invalid.env",
			wantCodes: []diagnostic.Code{
				diagnostic.CodeEnvEmpty,
				diagnostic.CodeEnvInvalidEnum,
				diagnostic.CodeEnvLengthOutOfRange,
				diagnostic.CodeEnvNumberOutOfRange,
				diagnostic.CodeEnvNumberOutOfRange,
				diagnostic.CodeEnvTypeMismatch,
			},
		},
		{
			name:     "nestjs api valid",
			contract: "nestjs.json",
			dotenv:   "nestjs-valid.env",
		},
		{
			name:     "frontend next.js public variables valid",
			contract: "frontend.json",
			dotenv:   "frontend-valid.env",
		},
		{
			name:     "secret-heavy backend valid",
			contract: "secret-heavy.json",
			dotenv:   "secret-heavy-valid.env",
		},
		{
			name:     "every value diagnostic",
			contract: "invalid-values.json",
			dotenv:   "invalid-values.env",
			wantCodes: []diagnostic.Code{
				diagnostic.CodeEnvEmpty,
				diagnostic.CodeEnvInvalidEnum,
				diagnostic.CodeEnvInvalidEnum,
				diagnostic.CodeEnvInvalidFormat,
				diagnostic.CodeEnvInvalidFormat,
				diagnostic.CodeEnvLengthOutOfRange,
				diagnostic.CodeEnvMissing,
				diagnostic.CodeEnvMultipleOf,
				diagnostic.CodeEnvNumberOutOfRange,
				diagnostic.CodeEnvNumberOutOfRange,
				diagnostic.CodeEnvPatternMismatch,
				diagnostic.CodeEnvTypeMismatch,
				diagnostic.CodeEnvTypeMismatch,
				diagnostic.CodeEnvTypeMismatch,
			},
		},
		{
			name:     "unknown variables are warnings",
			contract: "unknown-vars.json",
			dotenv:   "unknown-vars.env",
			wantCodes: []diagnostic.Code{
				diagnostic.CodeEnvUnknown,
				diagnostic.CodeEnvUnknown,
				diagnostic.CodeEnvUnknown,
			},
		},
		{
			name:     "ci process environment with unrelated entries",
			contract: "node-api.json",
			process:  "ci-server.env",
			wantCodes: []diagnostic.Code{
				diagnostic.CodeEnvUnknown,
				diagnostic.CodeEnvUnknown,
				diagnostic.CodeEnvUnknown,
				diagnostic.CodeEnvUnknown,
				diagnostic.CodeEnvUnknown,
				diagnostic.CodeEnvUnknown,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := loadContract(t, root, tc.contract)

			var env source.Environment
			switch {
			case tc.dotenv != "":
				env = loadDotenv(t, root, tc.dotenv)
			case tc.process != "":
				env = loadProcessEnvironment(t, root, tc.process)
			default:
				t.Fatal("scenario declares no environment source")
			}

			diags := validate.Validate(c, env)
			if !sameCodeMultiset(codesOf(diags), tc.wantCodes) {
				t.Errorf("diagnostic codes = %v, want %v", codesOf(diags), tc.wantCodes)
			}

			result := output.NewResult(diags)
			wantValid := len(tc.wantCodes) == 0 || containsOnlyWarnings(diags)
			if result.Valid() != wantValid {
				t.Errorf("Valid() = %v, want %v", result.Valid(), wantValid)
			}

			var human strings.Builder
			if err := output.WriteHuman(&human, result); err != nil {
				t.Fatalf("WriteHuman: %v", err)
			}
			var jsonOut strings.Builder
			if err := output.WriteJSON(&jsonOut, result); err != nil {
				t.Fatalf("WriteJSON: %v", err)
			}
			if strings.ContainsRune(human.String(), '\r') || strings.ContainsRune(jsonOut.String(), '\r') {
				t.Error("output contains a carriage return; must be LF-only")
			}
		})
	}
}

func codesOf(diags []diagnostic.Diagnostic) []diagnostic.Code {
	out := make([]diagnostic.Code, len(diags))
	for i, d := range diags {
		out[i] = d.Code
	}
	return out
}

func containsOnlyWarnings(diags []diagnostic.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == diagnostic.SeverityError {
			return false
		}
	}
	return true
}

func TestContractErrorScenarios(t *testing.T) {
	root := repoRoot(t)
	cases := []struct {
		name string
		file string
		want diagnostic.Code
	}{
		{"malformed json", "malformed.json", diagnostic.CodeContractInvalid},
		{"unsupported keyword", "unsupported-keyword.json", diagnostic.CodeContractUnsupportedKeyword},
		{"duplicate variable", "duplicate-variable.json", diagnostic.CodeContractDuplicateVariable},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(contractPath(root, tc.file))
			if err != nil {
				t.Fatalf("read contract fixture: %v", err)
			}
			c, diags := contract.Parse(data, tc.file)
			if c != nil {
				t.Fatalf("contract parsed despite expected %s", tc.want)
			}
			if len(diags) == 0 {
				t.Fatalf("expected at least one %s diagnostic, got none", tc.want)
			}
			for _, d := range diags {
				if d.Code != tc.want {
					t.Errorf("diagnostic code %s, want %s (%s)", d.Code, tc.want, tc.file)
				}
			}
		})
	}
}

func TestSourceErrorScenarios(t *testing.T) {
	root := repoRoot(t)

	// Malformed dotenv content must surface as a parse (ENV_FILE_INVALID-class)
	// error.
	_, err := source.LoadDotenv(filepath.Join(root, "testdata", "dotenv", "malformed.env"))
	if err == nil {
		t.Fatal("malformed.env parsed; want parse error")
	}
	if parsed := source.IsParseError(err); !parsed {
		t.Errorf("malformed.env error = %v; want parse error", err)
	}

	// A missing file is a read (SOURCE_INVALID-class) error, not a parse error.
	_, err = source.LoadDotenv(filepath.Join(root, "testdata", "dotenv", "does-not-exist.env"))
	if err == nil {
		t.Fatal("missing file loaded; want error")
	}
	if parsed := source.IsParseError(err); parsed {
		t.Errorf("missing file error = %v; want non-parse error", err)
	}
}

// TestDotenvUtf8AndCRLF proves that the same document parses identically with
// LF and CRLF line endings and that UTF-8 values are preserved byte-for-byte.
func TestDotenvUtf8AndCRLF(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "testdata", "dotenv", "unicode.env"))
	if err != nil {
		t.Fatalf("read unicode fixture: %v", err)
	}

	envLF, err := source.ParseDotenv(raw)
	if err != nil {
		t.Fatalf("parse LF: %v", err)
	}
	crlf := bytes.ReplaceAll(raw, []byte("\n"), []byte("\r\n"))
	envCRLF, err := source.ParseDotenv(crlf)
	if err != nil {
		t.Fatalf("parse CRLF: %v", err)
	}

	if !reflect.DeepEqual(envLF, envCRLF) {
		t.Errorf("LF and CRLF parsing differ:\nLF:   %#v\nCRLF: %#v", envLF, envCRLF)
	}
	want := map[string]string{
		"GREETING":   "héllo wörld — ünïcödé",
		"EMOJI":      "🚀 launch",
		"SITE_TITLE": "Zürich ✓",
	}
	if !reflect.DeepEqual(envLF, want) {
		t.Errorf("UTF-8 values were not preserved verbatim:\n got %#v\nwant %#v", envLF, want)
	}
}

// TestSecretsNeverSurfaceAndOutputDeterministic drives realistic secret
// values through full pipeline rendering and asserts they never appear in
// human or JSON output, and that output is byte-identical across runs.
func TestSecretsNeverSurfaceAndOutputDeterministic(t *testing.T) {
	root := repoRoot(t)
	secrets := []string{
		"CorrectHorseBatteryStaple",
		"s1gn1ng-5ecr3t-must-not-leak-00000000",
		"AKIAIOSFODNN7EXAMPLE",
		"c0mpl3xSMTPpassword!",
		"sk_live_51N3xt9ExAmPleToKen",
		"sk_prod_never_commit_this_key_42",
		"super-secret-password-for-signing-tokens",
		"postgres://app:secret@db.example.internal:5432/orders",
	}

	cases := []struct {
		contract string
		dotenv   string
	}{
		{"secret-heavy.json", "secret-heavy-valid.env"},
		{"fastapi.json", "fastapi-valid.env"},
		{"nestjs.json", "nestjs-valid.env"},
		{"invalid-values.json", "invalid-values.env"},
	}

	for _, tc := range cases {
		t.Run(tc.contract, func(t *testing.T) {
			c := loadContract(t, root, tc.contract)
			env := loadDotenv(t, root, tc.dotenv)
			result := output.NewResult(validate.Validate(c, env))

			render := func(sb *strings.Builder) {
				sb.Reset()
				if err := output.WriteHuman(sb, result); err != nil {
					t.Fatalf("WriteHuman: %v", err)
				}
			}
			var first strings.Builder
			render(&first)
			for i := 0; i < 3; i++ {
				var next strings.Builder
				render(&next)
				if next.String() != first.String() {
					t.Fatalf("human output is not deterministic:\nfirst:\n%s\nnext:\n%s", first.String(), next.String())
				}
			}

			var jsonFirst strings.Builder
			if err := output.WriteJSON(&jsonFirst, result); err != nil {
				t.Fatalf("WriteJSON: %v", err)
			}
			for i := 0; i < 3; i++ {
				var jsonNext strings.Builder
				if err := output.WriteJSON(&jsonNext, result); err != nil {
					t.Fatalf("WriteJSON: %v", err)
				}
				if jsonNext.String() != jsonFirst.String() {
					t.Fatal("JSON output is not deterministic")
				}
			}

			for _, secret := range secrets {
				if strings.Contains(first.String(), secret) {
					t.Errorf("secret %q leaked into human output:\n%s", secret, first.String())
				}
				if strings.Contains(jsonFirst.String(), secret) {
					t.Errorf("secret %q leaked into JSON output:\n%s", secret, jsonFirst.String())
				}
			}
		})
	}
}
