package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const exampleContract = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "PORT": {"type": "integer", "default": 8080, "description": "TCP port to listen on"},
    "DATABASE_URL": {"type": "string", "x-envdoctor-secret": true},
    "LOG_LEVEL": {"type": "string", "enum": ["debug", "info", "error"]}
  }
}`

func TestGenerateExampleWritesFile(t *testing.T) {
	dir := t.TempDir()
	contractPath := writeTempFile(t, dir, "c.json", exampleContract)
	outPath := filepath.Join(dir, ".env.example")

	code, stdout, stderr := runCLI(t, "generate", "example", "--contract", contractPath, "--output", outPath)
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Wrote") {
		t.Errorf("stdout = %q, want success notice", stdout)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	got := string(content)
	for _, want := range []string{
		"# TCP port to listen on\nPORT=8080",
		"DATABASE_URL=",
		"LOG_LEVEL=",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated file missing %q:\n%s", want, got)
		}
	}

	want := "DATABASE_URL=\nLOG_LEVEL=\n# TCP port to listen on\nPORT=8080\n"
	if got != want {
		t.Errorf("content mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestGenerateExampleDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	contractPath := writeTempFile(t, dir, "c.json", exampleContract)
	outPath := writeTempFile(t, dir, ".env.example", "KEEP=ME\n")

	code, _, stderr := runCLI(t, "generate", "example", "--contract", contractPath, "--output", outPath)
	if code != 2 {
		t.Fatalf("exit = %d, want 2; stderr=%q", code, stderr)
	}
	if !strings.Contains(stderr, "already exists") {
		t.Errorf("stderr = %q, want already-exists notice", stderr)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(content) != "KEEP=ME\n" {
		t.Errorf("existing file was modified: %q", content)
	}
}

func TestGenerateExampleUsesDefaultContractDiscovery(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, defaultContractFile, exampleContract)
	t.Chdir(dir)

	code, stdout, stderr := runCLI(t, "generate", "example")
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	content, err := os.ReadFile(filepath.Join(dir, defaultExampleFile))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	if !strings.Contains(string(content), "PORT=8080") {
		t.Errorf("generated file missing default:\n%s", content)
	}
}

func TestGenerateExampleMissingContractExit2(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "absent.schema.json")

	code, stdout, _ := runCLI(t, "generate", "example", "--contract", missing)
	if code != 2 {
		t.Fatalf("exit = %d, want 2; stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "CONTRACT_INVALID") {
		t.Errorf("stdout = %q, want CONTRACT_INVALID", stdout)
	}
}

func TestGenerateExampleMalformedContractExit2(t *testing.T) {
	dir := t.TempDir()
	contractPath := writeTempFile(t, dir, "c.json", "{ not json")

	code, stdout, _ := runCLI(t, "generate", "example", "--contract", contractPath)
	if code != 2 {
		t.Fatalf("exit = %d, want 2; stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "CONTRACT_INVALID") {
		t.Errorf("stdout = %q, want CONTRACT_INVALID", stdout)
	}
}

func TestGenerateUnknownSubcommand(t *testing.T) {
	code, _, stderr := runCLI(t, "generate")
	if code != 2 || !strings.Contains(stderr, "Usage: envdoctor generate example") {
		t.Errorf("generate with no subcommand: exit=%d stderr=%q", code, stderr)
	}
	code, _, stderr = runCLI(t, "generate", "frobnicate")
	if code != 2 || !strings.Contains(stderr, "unknown subcommand") {
		t.Errorf("generate frobnicate: exit=%d stderr=%q", code, stderr)
	}
}

func TestGenerateExampleHelp(t *testing.T) {
	code, stdout, _ := runCLI(t, "generate", "--help")
	if code != 0 || !strings.Contains(stdout, "Usage: envdoctor generate example") {
		t.Errorf("generate --help: exit=%d stdout=%q", code, stdout)
	}

	code, stdout, _ = runCLI(t, "generate", "example", "--help")
	if code != 0 || !strings.Contains(stdout, "Usage: envdoctor generate example") {
		t.Errorf("generate example --help: exit=%d stdout=%q", code, stdout)
	}
}

func TestGenerateExampleBadFlag(t *testing.T) {
	code, _, stderr := runCLI(t, "generate", "example", "--nope")
	if code != 2 || stderr == "" {
		t.Errorf("exit=%d stderr=%q", code, stderr)
	}
}

func TestGenerateExampleUnexpectedArgument(t *testing.T) {
	code, _, stderr := runCLI(t, "generate", "example", "extra")
	if code != 2 || !strings.Contains(stderr, "unexpected argument") {
		t.Errorf("exit=%d stderr=%q", code, stderr)
	}
}
