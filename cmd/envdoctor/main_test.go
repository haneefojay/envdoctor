package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--help"}, &stdout, &stderr); code != 0 { t.Fatalf("exit code = %d, want 0", code) }
	if !strings.Contains(stdout.String(), "EnvDoctor") { t.Fatal("help output is missing the product name") }
	if stderr.Len() != 0 { t.Fatal("help wrote to stderr") }
}

func TestUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"unknown"}, &stdout, &stderr); code != 2 { t.Fatalf("exit code = %d, want 2", code) }
}

func TestCheckReportsValidationWithoutSecret(t *testing.T) {
	directory := t.TempDir()
	contractPath := filepath.Join(directory, "envdoctor.schema.json")
	envPath := filepath.Join(directory, ".env")
	contract := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "x-envdoctor-version": "1",
  "type": "object",
  "properties": {
    "PORT": {"type": "integer", "minimum": 1},
    "TOKEN": {"type": "string", "x-envdoctor-secret": true}
  },
  "required": ["PORT", "TOKEN"]
}`
	if err := os.WriteFile(contractPath, []byte(contract), 0o600); err != nil { t.Fatal(err) }
	if err := os.WriteFile(envPath, []byte("PORT=not-a-number\nTOKEN=super-secret-password\n"), 0o600); err != nil { t.Fatal(err) }
	var stdout, stderr bytes.Buffer
	if code := run([]string{"check", "--contract", contractPath, "--env-file", envPath}, &stdout, &stderr); code != 1 { t.Fatalf("exit code = %d, want 1", code) }
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "ENV_TYPE_MISMATCH") { t.Fatal("type diagnostic missing") }
	if strings.Contains(combined, "super-secret-password") { t.Fatal("secret leaked in diagnostics") }
}

func TestGenerateExampleLeavesSecretBlank(t *testing.T) {
	directory := t.TempDir()
	contractPath := filepath.Join(directory, "envdoctor.schema.json")
	outputPath := filepath.Join(directory, ".env.example")
	contract := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "x-envdoctor-version": "1",
  "type": "object",
  "properties": {
    "PORT": {"type": "integer", "default": 3000},
    "TOKEN": {"type": "string", "x-envdoctor-secret": true}
  },
  "required": []
}`
	if err := os.WriteFile(contractPath, []byte(contract), 0o600); err != nil { t.Fatal(err) }
	var stdout, stderr bytes.Buffer
	if code := run([]string{"generate", "example", "--contract", contractPath, "--output", outputPath}, &stdout, &stderr); code != 0 { t.Fatalf("exit code = %d: %s", code, stderr.String()) }
	data, err := os.ReadFile(outputPath); if err != nil { t.Fatal(err) }
	if !strings.Contains(string(data), "PORT=3000") || !strings.Contains(string(data), "TOKEN=\n") { t.Fatalf("unexpected generated example: %q", string(data)) }
	if strings.Contains(string(data), "super-secret-password") { t.Fatal("secret appeared in generated example") }
}
