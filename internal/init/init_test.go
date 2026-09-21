package init

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/haneefojay/envdoctor/internal/contract"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverPrefersEnvExampleOverEnv(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, EnvExampleFile, "APP_NAME=example\nPORT=8080\n")
	writeFile(t, dir, EnvFile, "APP_NAME=prod\nPRIVATE_TOKEN=abc\n")

	names, source, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if source != EnvExampleFile {
		t.Errorf("source = %q, want %q", source, EnvExampleFile)
	}
	want := "APP_NAME,PORT"
	if got := strings.Join(names, ","); got != want {
		t.Errorf("names = %q, want %q", got, want)
	}

	repr, err := Template(names)
	if err != nil {
		t.Fatalf("Template: %v", err)
	}
	generated := string(repr)
	for _, forbidden := range []string{"example", "8080", "prod", "abc", "PRIVATE_TOKEN"} {
		if strings.Contains(generated, forbidden) {
			t.Errorf("generated contract leaks value or ignored variable %q", forbidden)
		}
	}
	if strings.Contains(generated, "secret") {
		t.Errorf("generated contract classifies something as a secret")
	}
}

func TestDiscoverFallsBackToEnv(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, EnvFile, "A=1\nB=2\n")

	names, source, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if source != EnvFile {
		t.Errorf("source = %q, want %q", source, EnvFile)
	}
	want := "A,B"
	if got := strings.Join(names, ","); got != want {
		t.Errorf("names = %q, want %q (deduplicated and sorted)", got, want)
	}
}

func TestDiscoverNoSource(t *testing.T) {
	dir := t.TempDir()

	if _, _, err := Discover(dir); err == nil {
		t.Fatal("Discover succeeded with no discovery source, want error")
	}
}

func TestDiscoverMalformedExampleIsError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, EnvExampleFile, "not a valid line\n")

	if _, _, err := Discover(dir); err == nil {
		t.Fatal("Discover ignored a malformed .env.example, want error")
	}
}

func TestTemplateIsValidDeterministicNeutral(t *testing.T) {
	names := []string{"APP_NAME", "PORT", "DATABASE_URL"}
	first, err := Template(names)
	if err != nil {
		t.Fatalf("Template: %v", err)
	}
	second, err := Template([]string{"DATABASE_URL", "PORT", "APP_NAME"})
	if err != nil {
		t.Fatalf("Template (reordered): %v", err)
	}
	if string(first) != string(second) {
		t.Fatal("Template output depends on input order")
	}
	if !strings.HasSuffix(string(first), "\n") {
		t.Fatal("Template output does not end in a newline")
	}
	parsed, diags := contract.Parse(first, "generated")
	if len(diags) != 0 {
		t.Fatalf("generated contract has diagnostics: %v", diags)
	}
	if len(parsed.Variables) != len(names) {
		t.Fatalf("parsed %d variables, want %d", len(parsed.Variables), len(names))
	}
	for _, name := range names {
		v := parsed.Variable(name)
		if v == nil {
			t.Errorf("generated contract missing %q", name)
			continue
		}
		if v.Type != contract.TypeString {
			t.Errorf("%s: type = %q, want %q", name, v.Type, contract.TypeString)
		}
		if v.Secret {
			t.Errorf("%s inferred as secret", name)
		}
		if v.HasDefault() {
			t.Errorf("%s inferred a default", name)
		}
		if v.Required {
			t.Errorf("%s inferred as required", name)
		}
	}
}

func TestTemplateEmpty(t *testing.T) {
	first, err := Template(nil)
	if err != nil {
		t.Fatalf("Template: %v", err)
	}
	if _, diags := contract.Parse(first, "generated"); len(diags) != 0 {
		t.Fatalf("empty template has diagnostics: %v", diags)
	}
}

func TestWriteSchemaCreatesAndRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "envdoctor.schema.json")

	if err := WriteSchema(path, []byte("{}")); err != nil {
		t.Fatalf("WriteSchema: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "{}" {
		t.Errorf("written content = %q, want %q", got, "{}")
	}

	if err := WriteSchema(path, []byte("overwritten")); err == nil {
		t.Fatal("WriteSchema overwrote an existing file, want error")
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "{}" {
		t.Errorf("existing file was modified: %q", got)
	}
}
