// Package security contains the P11 security-hardening guards: static,
// repo-wide checks over the shipping code that keep forbidden capabilities out
// of EnvDoctor.
//
// The guards enforce properties the product promises:
//
//   - no network access (no HTTP clients, raw dialing, or TLS);
//   - no shell execution or subprocess spawning, and no panic paths for
//     ordinary user/contract/source errors;
//   - no logging framework (log / slog) and no telemetry;
//   - no environment-dump mode outside the documented process source.
//
// Behavior-level coverage lives in the source, dotenv, validate, output,
// generate, init, and check tests (including cmd/envdoctor/security_test.go).
// These guards are belt-and-suspenders: they fail the build if a future change
// imports or calls a banned capability in non-test code.
package security

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// bannedImports are import paths that would enable forbidden capabilities in
// shipping code. net/url is allowed: it is parse-only and performs no I/O.
var bannedImports = []string{
	`"net/http"`,
	`"net/http/httptrace"`,
	`"crypto/tls"`,
	`"net"`,
	`"os/exec"`,
	`"log"`,
	`"log/slog"`,
}

// bannedTokens are calls and constructs forbidden in non-test code. They are
// scanned lexically as a second, independent layer under the import bans.
var bannedTokens = []string{
	"http.Get(",
	"http.Post(",
	"http.PostForm(",
	"http.NewRequest(",
	"http.DefaultClient",
	"net.Dial",
	"net.Listen",
	"os.StartProcess",
	"syscall.Exec",
	"exec.Command",
	"log.Print",
	"log.Printf",
	"log.Println",
	"log.Fatal",
	"panic(",
}

// telemetryMarkers are vendor/product identifiers for usage-tracking and
// external-instrumentation dependencies that must never enter the codebase.
var telemetryMarkers = []string{
	"telemetry",
	"sentry.",
	"segmentio/",
	"mixpanel",
	"amplitude",
	"newrelic.",
}

// repoRoot resolves the repository root from the test file's source path,
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

// nonTestSources returns every non-test Go file under cmd/ and internal/,
// excluding this package's own directory.
func nonTestSources(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	securityDir := filepath.Join(root, "internal", "security")
	for _, dir := range []string{"cmd", "internal"} {
		base := filepath.Join(root, dir)
		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if path == securityDir {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", base, err)
		}
	}
	return files
}

func TestNoBannedCapabilitiesInShippingCode(t *testing.T) {
	var violations []string
	for _, path := range nonTestSources(t, repoRoot(t)) {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		violations = append(violations, checkImports(path, src)...)
		violations = append(violations, checkTokens(path, src)...)
		violations = append(violations, checkTelemetry(path, src)...)
	}
	if len(violations) > 0 {
		t.Fatalf("forbidden capabilities in shipping code:\n%s", strings.Join(violations, "\n"))
	}
}

func checkImports(path string, src []byte) []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ImportsOnly)
	if err != nil {
		return []string{fmt.Sprintf("%s: cannot parse: %v", path, err)}
	}
	var out []string
	for _, spec := range f.Imports {
		imp := spec.Path.Value // includes quotes
		for _, banned := range bannedImports {
			if imp == banned {
				out = append(out, fmt.Sprintf("%s: forbids import %s", path, imp))
			}
		}
		if strings.HasPrefix(imp, `"golang.org/x/`) {
			out = append(out, fmt.Sprintf("%s: forbids external dependency %s", path, imp))
		}
	}
	return out
}

func checkTokens(path string, src []byte) []string {
	text := string(src)
	var out []string
	for _, tok := range bannedTokens {
		if strings.Contains(text, tok) {
			out = append(out, fmt.Sprintf("%s: forbids %q", path, tok))
		}
	}
	return out
}

func checkTelemetry(path string, src []byte) []string {
	lower := strings.ToLower(string(src))
	var out []string
	for _, marker := range telemetryMarkers {
		if strings.Contains(lower, marker) {
			out = append(out, fmt.Sprintf("%s: telemetry marker %q present", path, marker))
		}
	}
	return out
}

// TestEnvironDumpConfinedToProcessSource asserts that the only place shipping
// code reads the whole process environment is the documented process source.
// There is no generic environment-dump mode anywhere else.
func TestEnvironDumpConfinedToProcessSource(t *testing.T) {
	root := repoRoot(t)
	allowed := filepath.Join(root, "internal", "source", "process.go")
	var leaked []string
	for _, path := range nonTestSources(t, root) {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(src), "os.Environ()") && path != allowed {
			leaked = append(leaked, path)
		}
	}
	if len(leaked) > 0 {
		t.Fatalf("os.Environ() appears outside the process source boundary: %v", leaked)
	}
}
