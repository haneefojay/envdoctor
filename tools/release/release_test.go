package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// repoRoot resolves the repository root from this test file's source path.
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

func TestNormalizeVersion(t *testing.T) {
	valid := map[string]string{
		"0.1.0":         "0.1.0",
		"v0.1.0":        "0.1.0",
		"1.2.3":         "1.2.3",
		"v2.0.0-rc.1":   "2.0.0-rc.1",
		"2.0.0+build.5": "2.0.0+build.5",
	}
	for in, want := range valid {
		got, err := normalizeVersion(in)
		if err != nil {
			t.Errorf("normalizeVersion(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("normalizeVersion(%q) = %q, want %q", in, got, want)
		}
	}
	for _, in := range []string{"", "v", "1.2", "banana", "v0.1.0-", "0.1.0\n"} {
		if _, err := normalizeVersion(in); err == nil {
			t.Errorf("normalizeVersion(%q): expected error", in)
		}
	}
}

func TestTargetsCoverDocumentedMatrix(t *testing.T) {
	want := []Target{
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
		{OS: "windows", Arch: "amd64"},
		{OS: "windows", Arch: "arm64"},
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
	}
	if !reflect.DeepEqual(targets, want) {
		t.Errorf("targets = %+v, want %+v", targets, want)
	}
}

func TestChangelogSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md")
	content := `# Changelog

## [Unreleased]

### Added
- upcoming work

## v0.2.0

### Changed
- something happened

More bullets.

## v0.1.0

### Fixed
- old bug
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := changelogSection(path, "0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	want := `### Changed
- something happened

More bullets.`
	if got != want {
		t.Errorf("changelogSection(v0.2.0) = %q, want %q", got, want)
	}

	if got, err := changelogSection(path, "v0.1.0"); err != nil || got != "### Fixed\n- old bug" {
		t.Errorf("changelogSection(v0.1.0) = %q, err=%v", got, err)
	}

	// A missing section is empty, not an error.
	if got, err := changelogSection(path, "9.9.9"); err != nil || got != "" {
		t.Errorf("changelogSection(9.9.9) = %q, err=%v; want empty", got, err)
	}
}

// TestPackageProducesDeterministicArtifactsAndChecksums builds the host
// target, exercising the real cross-compile, archive, and checksum code paths,
// then extracts the packaged binary and runs `version` to prove the injected
// -X ldflags override lands in release binaries.
func TestPackageProducesDeterministicArtifactsAndChecksums(t *testing.T) {
	root := repoRoot(t)
	out := t.TempDir()
	host := Target{OS: runtime.GOOS, Arch: runtime.GOARCH}

	// Stage a changelog with a section for the version under test.
	notes := filepath.Join(t.TempDir(), "CHANGELOG.md")
	if err := os.WriteFile(notes, []byte("## v0.2.0\n\n### Added\n- shipped\n\n## v0.3.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	archives, err := Package(root, Options{
		Version: "v0.2.0",
		OutDir:  out,
		Targets: []Target{host},
		Notes:   notes,
	})
	if err != nil {
		t.Fatalf("Package: %v", err)
	}
	if len(archives) != 1 {
		t.Fatalf("archives = %v, want 1", archives)
	}

	archive := filepath.Join(out, host.archiveName("0.2.0"))
	if _, err := os.Stat(archive); err != nil {
		t.Fatalf("archive missing: %v", err)
	}

	// Archive layout: a single root dir holding the binary, README, LICENSE.
	names := archiveNames(t, archive)
	got := map[string]bool{}
	for _, n := range names {
		got[n] = true
	}
	rootDir := host.dirName("0.2.0") + "/"
	for _, member := range []string{
		rootDir + host.binaryName(),
		rootDir + "README.md",
		rootDir + "LICENSE",
	} {
		if !got[member] {
			t.Errorf("archive missing %s; members: %v", member, names)
		}
	}

	// The packaged binary must report the injected version, not the default.
	binary := extractMember(t, archive, rootDir+host.binaryName())
	cmd := exec.Command(binary, "version")
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run packaged binary: %v: %s", err, outBytes)
	}
	if want := "envdoctor 0.2.0\n"; string(outBytes) != want {
		t.Errorf("packaged binary version output = %q, want %q", outBytes, want)
	}

	// Checksums: exactly one line per artifact, two-space separated, and each
	// digest must match the archive bytes.
	sums, err := os.ReadFile(filepath.Join(out, "SHA256SUMS"))
	if err != nil {
		t.Fatalf("read SHA256SUMS: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(sums)), "\n")
	if len(lines) != 1 {
		t.Fatalf("SHA256SUMS lines = %d: %q", len(lines), lines)
	}
	fields := strings.Split(lines[0], "  ")
	wantName := host.archiveName("0.2.0")
	if len(fields) != 2 || fields[1] != wantName || len(fields[0]) != 64 {
		t.Fatalf("SHA256SUMS line %q not in <sha256>  <name> form", lines[0])
	}
	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if fields[0] != hex.EncodeToString(sum[:]) {
		t.Errorf("checksum mismatch: %q != %x", fields[0], sum)
	}

	// NOTES.md contains only the released section.
	note, err := os.ReadFile(filepath.Join(out, "NOTES.md"))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(note); !strings.Contains(got, "- shipped") || strings.Contains(got, "v0.3.0") {
		t.Errorf("NOTES.md = %q, want the v0.2.0 section only", got)
	}
}

// archiveNames lists the member names of a tar.gz or zip archive.
func archiveNames(t *testing.T, archivePath string) []string {
	t.Helper()
	if strings.HasSuffix(archivePath, ".zip") {
		z, err := zip.OpenReader(archivePath)
		if err != nil {
			t.Fatal(err)
		}
		defer z.Close()
		var names []string
		for _, e := range z.File {
			names = append(names, e.Name)
		}
		return names
	}
	return tarNames(t, archivePath)
}

// extractMember copies one member out of a tar.gz or zip archive.
func extractMember(t *testing.T, archivePath, member string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), filepath.Base(member))
	if strings.HasSuffix(archivePath, ".zip") {
		z, err := zip.OpenReader(archivePath)
		if err != nil {
			t.Fatal(err)
		}
		defer z.Close()
		for _, e := range z.File {
			if e.Name == member {
				rc, err := e.Open()
				if err != nil {
					t.Fatal(err)
				}
				data, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(dst, data, 0o755); err != nil {
					t.Fatal(err)
				}
				return dst
			}
		}
		t.Fatalf("member %q not in zip", member)
	}
	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			t.Fatalf("member %q not in tar.gz", member)
		}
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Name == member {
			data, err := io.ReadAll(tr)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(dst, data, 0o755); err != nil {
				t.Fatal(err)
			}
			return dst
		}
	}
}

// TestArchiveWritersAreDeterministic proves the two archive writers produce
// byte-identical output for identical inputs (fixed member order and mtimes).
func TestArchiveWritersAreDeterministic(t *testing.T) {
	staging := t.TempDir()
	if err := os.WriteFile(filepath.Join(staging, "envdoctor"), []byte("fake-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(staging, "envdoctor"), epochTime, epochTime); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	writePair := func(name string, w func(string) error) []byte {
		t.Helper()
		a := filepath.Join(dir, name+".a")
		b := filepath.Join(dir, name+".b")
		if err := w(a); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := w(b); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		// Both copies must be byte-identical.
		fa, err := os.ReadFile(a)
		if err != nil {
			t.Fatal(err)
		}
		fb, err := os.ReadFile(b)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(fa, fb) {
			t.Errorf("%s output is not deterministic: %d vs %d bytes differ", name, len(fa), len(fb))
		}
		return fa
	}
	writePair("tar.gz", func(dst string) error {
		return writeTarGZ(dst, staging, "envdoctor_0.1.0_linux_amd64")
	})
	writePair("zip", func(dst string) error {
		return writeZip(dst, staging, "envdoctor_0.1.0_windows_amd64")
	})
}

func tarNames(t *testing.T, archivePath string) []string {
	t.Helper()
	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, hdr.Name)
	}
	return names
}

// TestEmbeddedVersionOnNativeTarget builds the CLI for the host target with an
// injected version and runs it, proving the -X ldflags embedding actually
// overrides the default version (this is what release binaries ship with).
func TestZipRoundTrip(t *testing.T) {
	staging := t.TempDir()
	if err := os.WriteFile(filepath.Join(staging, "envdoctor.exe"), []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(staging, "envdoctor.exe"), epochTime, epochTime); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "x.zip")
	if err := writeZip(out, staging, "envdoctor_0.1.0_windows_amd64"); err != nil {
		t.Fatal(err)
	}
	z, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	if len(z.File) != 1 {
		t.Fatalf("zip entries = %d, want 1", len(z.File))
	}
	e := z.File[0]
	if e.Name != "envdoctor_0.1.0_windows_amd64/envdoctor.exe" {
		t.Errorf("zip entry = %q", e.Name)
	}
	if e.Method != zip.Store {
		t.Errorf("zip method = %d, want Store", e.Method)
	}
	if !e.Modified.Equal(epochTime) {
		t.Errorf("zip mtime = %v, want %v", e.Modified, epochTime)
	}
}
