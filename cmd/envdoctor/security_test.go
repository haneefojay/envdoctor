package main

// P11 security-hardening tests. These run the real CLI surface end to end with
// deliberately realistic, distinctive secret values and assert that no secret
// ever reaches stdout, stderr, or a generated file. Each failing variable
// exercises a different diagnostic path, so a value that embeds itself in any
// future message, JSON field, or error would be caught here.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// p11SecretContract declares eleven secret variables whose values force one
// value-driven diagnostic each (plus two that pass). Every secret value must
// remain absent from all output surfaces even while its variable fails.
const p11SecretContract = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "SECRET_STR":     {"type": "string",  "minLength": 200, "x-envdoctor-secret": true},
    "SECRET_INT":     {"type": "integer", "x-envdoctor-secret": true},
    "SECRET_ENUM":    {"type": "string",  "enum": ["alpha", "beta"], "x-envdoctor-secret": true},
    "SECRET_PATTERN": {"type": "string",  "pattern": "^[A-Z0-9]+$", "x-envdoctor-secret": true},
    "SECRET_EMAIL":   {"type": "string",  "format": "email", "x-envdoctor-secret": true},
    "SECRET_NUM":     {"type": "number",  "maximum": 1000, "x-envdoctor-secret": true},
    "SECRET_MULT":    {"type": "integer", "multipleOf": 7, "x-envdoctor-secret": true},
    "SECRET_QUOTE":   {"type": "string",  "minLength": 200, "x-envdoctor-secret": true},
    "SECRET_NEWLINE": {"type": "string",  "minLength": 200, "x-envdoctor-secret": true},
    "SECRET_BOOL":    {"type": "boolean", "x-envdoctor-secret": true},
    "SECRET_PASS":    {"type": "string",  "x-envdoctor-secret": true}
  }
}`

// TestCheckNeverDisclosesRealisticSecrets drives every value-driven diagnostic
// with a realistic secret value and asserts the decoded secret is absent from
// both human and JSON output on stdout and from stderr. Values containing
// double quotes and newlines are written through the documented dotenv escapes
// so the decoded values exercise the exact strings that leave the parser.
func TestCheckNeverDisclosesRealisticSecrets(t *testing.T) {
	dir := t.TempDir()
	contractPath := writeTempFile(t, dir, "c.json", p11SecretContract)
	envPath := writeTempFile(t, dir, ".env", strings.Join([]string{
		"SECRET_STR=" + "super-secret-password",
		"SECRET_INT=" + "postgres://user:password@host:5432/db",
		"SECRET_ENUM=" + "ghp_9zYxWvUtSrQpOnMlKjIhGfEdCbA0123456789",
		"SECRET_PATTERN=" + "8f14e45fceea167a5a36dedd4bea2543",
		"SECRET_EMAIL=" + "prod-token with spaces",
		"SECRET_NUM=" + "999999999999",
		"SECRET_MULT=" + "13",
		`SECRET_QUOTE="after\"quote"`,
		`SECRET_NEWLINE="line1\nline2"`,
		"SECRET_BOOL=" + "true",
		"SECRET_PASS=" + "s3cr3t-t0k3n-9zYxWv",
	}, "\n")+"\n")

	// The exact decoded secret values. The two newline/quote encodings above
	// decode to the literal values listed here.
	secrets := []string{
		"super-secret-password",
		"postgres://user:password@host:5432/db",
		"ghp_9zYxWvUtSrQpOnMlKjIhGfEdCbA0123456789",
		"8f14e45fceea167a5a36dedd4bea2543",
		"prod-token with spaces",
		"999999999999",
		"13",
		`after"quote`,
		"line1\nline2",
		"s3cr3t-t0k3n-9zYxWv",
	}

	for _, mode := range []string{"", "--json"} {
		args := []string{"check", "--contract", contractPath, "--env-file", envPath}
		if mode != "" {
			args = append(args, mode)
		}
		code, stdout, stderr := runCLI(t, args...)
		if code != 1 {
			t.Fatalf("mode %q: exit = %d, want 1. stdout=%q stderr=%q", mode, code, stdout, stderr)
		}
		if mode != "" && !json.Valid([]byte(stdout)) {
			t.Fatalf("mode %q: stdout is not valid JSON: %q", mode, stdout)
		}
		for _, secret := range secrets {
			if strings.Contains(stdout, secret) || strings.Contains(stderr, secret) {
				t.Fatalf("mode %q leaked secret value %q", mode, secret)
			}
		}
	}
}

// TestCheckMalformedEnvFileEchoesNoSecret ensures a file that fails parsing does
// not echo already-parsed secret values or the raw source text into output: the
// whole dotenv document is rejected, never partially reported.
func TestCheckMalformedEnvFileEchoesNoSecret(t *testing.T) {
	const secret = "super-secret-password"
	dir := t.TempDir()
	contractPath := writeTempFile(t, dir, "c.json", minimalContract)
	envPath := writeTempFile(t, dir, ".env", "DATABASE_URL="+secret+"\nTOKEN=$HOME\n")

	code, stdout, stderr := runCLI(t, "check", "--contract", contractPath, "--env-file", envPath)
	if code != 2 {
		t.Fatalf("exit = %d, want 2; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "ENV_FILE_INVALID") {
		t.Errorf("stdout = %q, want ENV_FILE_INVALID", stdout)
	}
	for _, forbidden := range []string{secret, "$HOME"} {
		if strings.Contains(stdout, forbidden) || strings.Contains(stderr, forbidden) {
			t.Fatalf("output leaked %q", forbidden)
		}
	}
}

// TestInitNeverCopiesRealisticSecrets verifies envdoctor init through the CLI:
// it discovers the secret-bearing variable names but copies no values, and
// neither the success output nor the written contract discloses a secret.
func TestInitNeverCopiesRealisticSecrets(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, ".env", strings.Join([]string{
		"API_KEY=super-secret-password",
		"DATABASE_URL=postgres://user:password@host:5432/db",
		"GITHUB_TOKEN=ghp_9zYxWvUtSrQpOnMlKjIhGfEdCbA0123456789",
		"APP_NAME=MyService",
		"PORT=8080",
	}, "\n")+"\n")
	t.Chdir(dir)

	code, stdout, stderr := runCLI(t, "init")
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	content, err := os.ReadFile(filepath.Join(dir, "envdoctor.schema.json"))
	if err != nil {
		t.Fatalf("read generated contract: %v", err)
	}
	for _, secret := range []string{
		"super-secret-password",
		"user:password",
		"ghp_9zYxWvUtSrQpOnMlKjIhGfEdCbA0123456789",
		"MyService",
		"8080",
	} {
		if strings.Contains(stdout+stderr, secret) {
			t.Fatalf("stdout or stderr leaked value %q", secret)
		}
		if strings.Contains(string(content), secret) {
			t.Fatalf("generated contract leaked value %q", secret)
		}
	}
	// The secret variable names themselves must be present (names are not
	// secrets and belong in the contract for review).
	for _, name := range []string{"API_KEY", "DATABASE_URL", "GITHUB_TOKEN"} {
		if !strings.Contains(string(content), name) {
			t.Errorf("generated contract missing discovered variable %q", name)
		}
	}
}

// TestGenerateExampleBlankForSecretsE2E verifies the generate command writes
// blank values for secret variables and cannot copy any environment value into
// the generated file.
func TestGenerateExampleBlankForSecretsE2E(t *testing.T) {
	dir := t.TempDir()
	contractPath := writeTempFile(t, dir, "c.json", `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "APP_NAME":     {"type": "string", "default": "MyService"},
    "API_KEY":      {"type": "string", "x-envdoctor-secret": true},
    "DATABASE_URL": {"type": "string", "x-envdoctor-secret": true}
  }
}`)
	outPath := filepath.Join(dir, ".env.example")

	code, stdout, stderr := runCLI(t, "generate", "example", "--contract", contractPath, "--output", outPath)
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	if !strings.Contains(string(content), "APP_NAME=MyService\n") {
		t.Errorf("generated file should carry the declared non-secret default:\n%s", content)
	}
	for _, name := range []string{"API_KEY", "DATABASE_URL"} {
		if !strings.Contains(string(content), name+"=\n") {
			t.Errorf("secret variable %s must be blank in the generated file:\n%s", name, content)
		}
	}
	for _, secret := range []string{"super-secret-password", "user:password"} {
		if strings.Contains(stdout+stderr+string(content), secret) {
			t.Fatalf("secret value %q leaked", secret)
		}
	}
}
