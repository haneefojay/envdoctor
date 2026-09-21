package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/haneefojay/envdoctor/internal/contract"
)

const validContract = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["DATABASE_URL"],
  "properties": {
    "DATABASE_URL": {"type": "string"},
    "PORT": {"type": "integer", "minimum": 1, "maximum": 65535},
    "DEBUG": {"type": "boolean"}
  }
}`

const minimalContract = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["DATABASE_URL"],
  "properties": {"DATABASE_URL": {"type": "string"}}
}`

const secretContract = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["TOKEN"],
  "properties": {
    "TOKEN": {"type": "string", "minLength": 100, "x-envdoctor-secret": true}
  }
}`

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func runCLI(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

type cliJSON struct {
	Valid       bool `json:"valid"`
	Diagnostics []struct {
		Severity string `json:"severity"`
		Code     string `json:"code"`
		Variable string `json:"variable"`
		Message  string `json:"message"`
	} `json:"diagnostics"`
}

func parseJSON(t *testing.T, out string) cliJSON {
	t.Helper()
	if strings.Contains(out, "\x1b") {
		t.Fatalf("JSON output contains ANSI escapes: %q", out)
	}
	var doc cliJSON
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	return doc
}

func TestNoArgsIsUsageError(t *testing.T) {
	code, stdout, stderr := runCLI(t)
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "Usage:") || !strings.Contains(stderr, "check") {
		t.Errorf("stderr = %q, want usage", stderr)
	}
}

func TestUnknownCommandIsUsageError(t *testing.T) {
	code, _, stderr := runCLI(t, "frobnicate")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr = %q, want unknown-command notice", stderr)
	}
}

func TestVersionAndHelpCommands(t *testing.T) {
	code, stdout, _ := runCLI(t, "version")
	if code != 0 || !strings.Contains(stdout, "envdoctor "+version) {
		t.Errorf("version: exit=%d stdout=%q", code, stdout)
	}

	code, stdout, _ = runCLI(t, "help")
	if code != 0 || !strings.Contains(stdout, "Usage:") {
		t.Errorf("help: exit=%d stdout=%q", code, stdout)
	}
}

func TestCheckHelpGoesToStdout(t *testing.T) {
	code, stdout, stderr := runCLI(t, "check", "--help")
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "Usage: envdoctor check") {
		t.Errorf("stdout = %q, want check usage", stdout)
	}
}

func TestCheckBadFlag(t *testing.T) {
	code, _, stderr := runCLI(t, "check", "--nope")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if stderr == "" {
		t.Errorf("stderr empty, want parse error")
	}
}

func TestCheckUnexpectedArgument(t *testing.T) {
	code, _, stderr := runCLI(t, "check", "extra")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(stderr, "unexpected argument") {
		t.Errorf("stderr = %q, want unexpected-argument notice", stderr)
	}
}

func TestCheckValidFile(t *testing.T) {
	dir := t.TempDir()
	contractPath := writeTempFile(t, dir, "envdoctor.schema.json", validContract)
	envPath := writeTempFile(t, dir, ".env", "DATABASE_URL=postgres://localhost/db\nPORT=3000\nDEBUG=true\n")

	code, stdout, stderr := runCLI(t, "check", "--contract", contractPath, "--env-file", envPath)
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Environment is valid") {
		t.Errorf("stdout = %q, want validity notice", stdout)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

func TestCheckMissingRequiredExit1(t *testing.T) {
	dir := t.TempDir()
	contractPath := writeTempFile(t, dir, "c.json", validContract)
	envPath := writeTempFile(t, dir, ".env", "PORT=3000\nDEBUG=true\n")

	code, stdout, _ := runCLI(t, "check", "--contract", contractPath, "--env-file", envPath)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "ENV_MISSING") || !strings.Contains(stdout, "DATABASE_URL") {
		t.Errorf("stdout = %q, want ENV_MISSING for DATABASE_URL", stdout)
	}
}

func TestCheckTypeMismatchExit1(t *testing.T) {
	dir := t.TempDir()
	contractPath := writeTempFile(t, dir, "c.json", validContract)
	envPath := writeTempFile(t, dir, ".env", "DATABASE_URL=postgres://localhost/db\nPORT=notanumber\n")

	code, stdout, _ := runCLI(t, "check", "--contract", contractPath, "--env-file", envPath)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "ENV_TYPE_MISMATCH") {
		t.Errorf("stdout = %q, want ENV_TYPE_MISMATCH", stdout)
	}
}

func TestCheckUnknownVariableIsWarning(t *testing.T) {
	dir := t.TempDir()
	contractPath := writeTempFile(t, dir, "c.json", minimalContract)
	envPath := writeTempFile(t, dir, ".env", "DATABASE_URL=x\nEXTRA=1\n")

	code, stdout, _ := runCLI(t, "check", "--contract", contractPath, "--env-file", envPath)
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "ENV_UNKNOWN") || !strings.Contains(stdout, "with warnings") {
		t.Errorf("stdout = %q, want unknown-variable warning", stdout)
	}
}

func TestCheckJSONValid(t *testing.T) {
	dir := t.TempDir()
	contractPath := writeTempFile(t, dir, "c.json", validContract)
	envPath := writeTempFile(t, dir, ".env", "DATABASE_URL=postgres://localhost/db\n")

	code, stdout, _ := runCLI(t, "check", "--contract", contractPath, "--env-file", envPath, "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stdout=%q", code, stdout)
	}
	doc := parseJSON(t, stdout)
	if !doc.Valid || len(doc.Diagnostics) != 0 {
		t.Errorf("doc = %+v, want valid with no diagnostics", doc)
	}
}

func TestCheckJSONInvalid(t *testing.T) {
	dir := t.TempDir()
	contractPath := writeTempFile(t, dir, "c.json", validContract)
	envPath := writeTempFile(t, dir, ".env", "PORT=1\n")

	code, stdout, _ := runCLI(t, "check", "--contract", contractPath, "--env-file", envPath, "--json")
	if code != 1 {
		t.Fatalf("exit = %d, want 1; stdout=%q", code, stdout)
	}
	doc := parseJSON(t, stdout)
	if doc.Valid {
		t.Errorf("valid = true, want false")
	}
	if len(doc.Diagnostics) == 0 || doc.Diagnostics[0].Code != "ENV_MISSING" {
		t.Errorf("diagnostics = %+v, want ENV_MISSING", doc.Diagnostics)
	}
}

func TestCheckContractMissingExit2(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "absent.schema.json")

	code, stdout, _ := runCLI(t, "check", "--contract", missing)
	if code != 2 {
		t.Fatalf("exit = %d, want 2; stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "CONTRACT_INVALID") {
		t.Errorf("stdout = %q, want CONTRACT_INVALID", stdout)
	}
}

func TestCheckContractMalformedExit2(t *testing.T) {
	dir := t.TempDir()
	contractPath := writeTempFile(t, dir, "c.json", "{ not json")

	code, stdout, _ := runCLI(t, "check", "--contract", contractPath)
	if code != 2 {
		t.Fatalf("exit = %d, want 2; stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "CONTRACT_INVALID") {
		t.Errorf("stdout = %q, want CONTRACT_INVALID", stdout)
	}
}

func TestCheckEnvFileMissingExit2(t *testing.T) {
	dir := t.TempDir()
	contractPath := writeTempFile(t, dir, "c.json", minimalContract)
	missing := filepath.Join(dir, "absent.env")

	code, stdout, _ := runCLI(t, "check", "--contract", contractPath, "--env-file", missing)
	if code != 2 {
		t.Fatalf("exit = %d, want 2; stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "SOURCE_INVALID") {
		t.Errorf("stdout = %q, want SOURCE_INVALID", stdout)
	}
}

func TestCheckEnvFileMalformedExit2(t *testing.T) {
	dir := t.TempDir()
	contractPath := writeTempFile(t, dir, "c.json", minimalContract)
	envPath := writeTempFile(t, dir, ".env", "DATABASE_URL=$HOME\n")

	code, stdout, _ := runCLI(t, "check", "--contract", contractPath, "--env-file", envPath)
	if code != 2 {
		t.Fatalf("exit = %d, want 2; stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "ENV_FILE_INVALID") {
		t.Errorf("stdout = %q, want ENV_FILE_INVALID", stdout)
	}
	if strings.Contains(stdout, "$HOME") {
		t.Errorf("stdout leaked source content: %q", stdout)
	}
}

func TestCheckMutuallyExclusiveSources(t *testing.T) {
	code, _, stderr := runCLI(t, "check", "--env-file", "x.env", "--environment")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(stderr, "choose one") {
		t.Errorf("stderr = %q, want source-selection error", stderr)
	}
}

func TestCheckProcessEnvironment(t *testing.T) {
	dir := t.TempDir()
	contractPath := writeTempFile(t, dir, "c.json", minimalContract)
	t.Setenv("DATABASE_URL", "postgres://localhost/db")

	code, stdout, _ := runCLI(t, "check", "--contract", contractPath, "--environment")
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "Environment is valid") {
		t.Errorf("stdout = %q, want validity notice", stdout)
	}
}

func TestCheckDefaultContractDiscovery(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, defaultContractFile, minimalContract)
	writeTempFile(t, dir, ".env", "DATABASE_URL=x\n")
	t.Chdir(dir)

	code, stdout, stderr := runCLI(t, "check", "--env-file", ".env")
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Environment is valid") {
		t.Errorf("stdout = %q, want validity notice", stdout)
	}
}

func TestCheckNeverLeaksSecretValue(t *testing.T) {
	const secret = "supersecretvalue-0xDEADBEEF-1234567890"
	dir := t.TempDir()
	contractPath := writeTempFile(t, dir, "c.json", secretContract)
	envPath := writeTempFile(t, dir, ".env", "TOKEN="+secret+"\n")

	for _, mode := range []string{"", "--json"} {
		args := []string{"check", "--contract", contractPath, "--env-file", envPath}
		if mode != "" {
			args = append(args, mode)
		}
		code, stdout, stderr := runCLI(t, args...)
		if code != 1 {
			t.Fatalf("mode %q: exit = %d, want 1", mode, code)
		}
		if strings.Contains(stdout, secret) || strings.Contains(stderr, secret) {
			t.Fatalf("mode %q leaked secret value", mode)
		}
	}
}

func TestInitWritesDraftContract(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, ".env.example", "APP_NAME=example\nPORT=8080\nDATABASE_URL=postgres://localhost/db\n")
	writeTempFile(t, dir, ".env", "APP_NAME=prod\nPRIVATE_TOKEN=abc\n")
	t.Chdir(dir)

	code, stdout, stderr := runCLI(t, "init")
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Wrote") || !strings.Contains(stdout, "3 variables") {
		t.Errorf("stdout = %q, want write notice", stdout)
	}

	pending := filepath.Join(dir, "envdoctor.schema.json")
	content, err := os.ReadFile(pending)
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}
	for _, forbidden := range []string{"example", "8080", "postgres", "prod", "abc"} {
		if strings.Contains(string(content), forbidden) {
			t.Errorf("contract leaks discovered value %q", forbidden)
		}
	}
	parsed, diags := contract.Parse(content, "generated")
	if len(diags) != 0 {
		t.Fatalf("generated contract has diagnostics: %v", diags)
	}
	if len(parsed.Variables) != 3 {
		t.Fatalf("parsed %d variables, want 3", len(parsed.Variables))
	}
	for _, name := range []string{"APP_NAME", "DATABASE_URL", "PORT"} {
		v := parsed.Variable(name)
		if v == nil {
			t.Errorf("missing %q", name)
			continue
		}
		if v.Type != contract.TypeString {
			t.Errorf("%s: type=%q, want string", name, v.Type)
		}
		if v.Secret {
			t.Errorf("%s inferred as secret", name)
		}
	}
}
