// Package main implements the EnvDoctor release packager: it builds the CLI
// for every documented target, wraps each binary in a deterministic archive,
// and writes a SHA256SUMS checksum manifest (plus, optionally, the CHANGELOG
// section for the released version).
//
// This is a development/build tool for tagged releases (used by
// .github/workflows/release.yml), not part of the shipped EnvDoctor product.
// It deliberately lives under tools/ rather than cmd/ or internal/ so the
// shipping-code security guards (which scan cmd/ and internal/) are not
// weakened by its use of `go build` as a subprocess.
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// versionSymbol is the linker symbol of the CLI's version variable. The main
// package's symbols are registered under the package name "main", so -X must
// target "main.version" regardless of the package's canonical import path.
const versionSymbol = "main.version"

// epochTime is the fixed timestamp stamped on every archive member so that
// archives are byte-for-byte reproducible for identical inputs, independent
// of when the release is built.
var epochTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// Target is one GOOS/GOARCH combination EnvDoctor releases for.
type Target struct {
	OS   string
	Arch string
}

// targets is the documented release matrix, in deterministic order.
var targets = []Target{
	{OS: "linux", Arch: "amd64"},
	{OS: "linux", Arch: "arm64"},
	{OS: "windows", Arch: "amd64"},
	{OS: "windows", Arch: "arm64"},
	{OS: "darwin", Arch: "amd64"},
	{OS: "darwin", Arch: "arm64"},
}

func (t Target) binaryName() string {
	if t.OS == "windows" {
		return "envdoctor.exe"
	}
	return "envdoctor"
}

func (t Target) dirName(version string) string {
	return fmt.Sprintf("envdoctor_%s_%s_%s", version, t.OS, t.Arch)
}

func (t Target) archiveName(version string) string {
	if t.OS == "windows" {
		return t.dirName(version) + ".zip"
	}
	return t.dirName(version) + ".tar.gz"
}

var semverRE = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)

// normalizeVersion validates a semantic version and strips a leading "v" so
// that embedded version strings and artifact names use MAJOR.MINOR.PATCH.
func normalizeVersion(version string) (string, error) {
	if !semverRE.MatchString(version) {
		return "", fmt.Errorf("version %q is not a semantic version (vMAJOR.MINOR.PATCH)", version)
	}
	return strings.TrimPrefix(version, "v"), nil
}

// Options configures Package.
type Options struct {
	// Version is the semantic version to embed and use in artifact names.
	// A leading "v" is stripped.
	Version string
	// OutDir is the directory for archives and SHA256SUMS.
	OutDir string
	// Targets restricts packaging to the given targets; nil packages all.
	Targets []Target
	// Notes, when non-empty, is a CHANGELOG file whose section for
	// Version is written to <OutDir>/NOTES.md.
	Notes string
}

// Package builds and archives every requested target and writes the
// SHA256SUMS manifest. It returns the absolute paths of the produced
// archives.
func Package(moduleRoot string, opts Options) ([]string, error) {
	version, err := normalizeVersion(opts.Version)
	if err != nil {
		return nil, err
	}
	if opts.OutDir == "" {
		return nil, fmt.Errorf("output directory is required")
	}
	outAbs, err := filepath.Abs(opts.OutDir)
	if err != nil {
		return nil, fmt.Errorf("resolve output directory: %w", err)
	}
	if err := os.MkdirAll(outAbs, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	ts := opts.Targets
	if ts == nil {
		ts = targets
	}
	if len(ts) == 0 {
		return nil, fmt.Errorf("no targets to package")
	}

	var archives []string
	for _, target := range ts {
		archive, err := packageTarget(moduleRoot, outAbs, target, version)
		if err != nil {
			return nil, err
		}
		archives = append(archives, archive)
	}

	if err := writeChecksums(outAbs, archives); err != nil {
		return nil, err
	}

	if opts.Notes != "" {
		section, err := changelogSection(opts.Notes, version)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(outAbs, "NOTES.md"), []byte(section), 0o644); err != nil {
			return nil, fmt.Errorf("write notes: %w", err)
		}
	}
	return archives, nil
}

// packageTarget builds one target into a staging directory, copies the
// repository README and LICENSE alongside the binary, and archives it.
func packageTarget(moduleRoot, outAbs string, target Target, version string) (string, error) {
	staging, err := os.MkdirTemp(os.TempDir(), "envdoctor-release-*")
	if err != nil {
		return "", fmt.Errorf("create staging directory: %w", err)
	}
	defer os.RemoveAll(staging)

	binary := filepath.Join(staging, target.binaryName())
	if err := buildBinary(moduleRoot, target, version, binary); err != nil {
		return "", err
	}
	for _, name := range []string{"README.md", "LICENSE"} {
		if err := copyIntoStaging(moduleRoot, staging, name); err != nil {
			return "", err
		}
	}
	if err := os.Chmod(binary, 0o755); err != nil {
		return "", fmt.Errorf("chmod binary: %w", err)
	}

	archive := filepath.Join(outAbs, target.archiveName(version))
	if target.OS == "windows" {
		if err := writeZip(archive, staging, target.dirName(version)); err != nil {
			return "", err
		}
	} else {
		if err := writeTarGZ(archive, staging, target.dirName(version)); err != nil {
			return "", err
		}
	}
	return archive, nil
}

// buildBinary cross-compiles the CLI for target with the version embedded via
// ldflags. CGO is disabled so the binaries link without a C toolchain.
func buildBinary(moduleRoot string, target Target, version, binaryDest string) error {
	cmd := exec.Command("go", "build",
		"-trimpath",
		"-ldflags", "-X "+versionSymbol+"="+version,
		"-o", binaryDest,
		"./cmd/envdoctor")
	cmd.Dir = moduleRoot
	cmd.Env = append(os.Environ(),
		"GOOS="+target.OS,
		"GOARCH="+target.Arch,
		"CGO_ENABLED=0",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("build %s/%s: %w: %s", target.OS, target.Arch, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// copyIntoStaging copies repoFile from moduleRoot into staging with a fixed
// permission and mtime so archives stay reproducible.
func copyIntoStaging(moduleRoot, staging, repoFile string) error {
	src, err := os.Open(filepath.Join(moduleRoot, repoFile))
	if err != nil {
		return fmt.Errorf("open %s: %w", repoFile, err)
	}
	defer src.Close()
	dst, err := os.OpenFile(filepath.Join(staging, repoFile), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create %s: %w", repoFile, err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return fmt.Errorf("copy %s: %w", repoFile, err)
	}
	if err := dst.Close(); err != nil {
		return err
	}
	return os.Chtimes(filepath.Join(staging, repoFile), epochTime, epochTime)
}

func writeTarGZ(archivePath, staging, rootDir string) error {
	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	entries, err := stagedEntries(staging)
	if err != nil {
		return err
	}
	for _, rel := range entries {
		info, err := os.Stat(filepath.Join(staging, rel))
		if err != nil {
			return err
		}
		hdr := &tar.Header{
			Name:       filepath.ToSlash(filepath.Join(rootDir, rel)),
			Mode:       int64(info.Mode().Perm()),
			Size:       info.Size(),
			ModTime:    epochTime,
			AccessTime: epochTime,
			ChangeTime: epochTime,
			Format:     tar.FormatPAX,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		src, err := os.Open(filepath.Join(staging, rel))
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, src); err != nil {
			src.Close()
			return err
		}
		if err := src.Close(); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	return f.Close()
}

func writeZip(archivePath, staging, rootDir string) error {
	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	entries, err := stagedEntries(staging)
	if err != nil {
		return err
	}
	for _, rel := range entries {
		src, err := os.Open(filepath.Join(staging, rel))
		if err != nil {
			return err
		}
		info, err := src.Stat()
		if err != nil {
			src.Close()
			return err
		}
		hdr := &zip.FileHeader{
			Name:     filepath.ToSlash(filepath.Join(rootDir, rel)),
			Method:   zip.Store,
			Modified: epochTime,
		}
		hdr.SetMode(info.Mode().Perm())
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			src.Close()
			return err
		}
		if _, err := io.Copy(w, src); err != nil {
			src.Close()
			return err
		}
		if err := src.Close(); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return f.Close()
}

// stagedEntries lists the files inside staging relative to staging, sorted
// for deterministic output.
func stagedEntries(staging string) ([]string, error) {
	var entries []string
	err := filepath.WalkDir(staging, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(staging, path)
		if err != nil {
			return err
		}
		entries = append(entries, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(entries)
	return entries, nil
}

func writeChecksums(dir string, archives []string) error {
	names := make([]string, 0, len(archives))
	for _, a := range archives {
		names = append(names, filepath.Base(a))
	}
	sort.Strings(names)

	var sb strings.Builder
	for _, name := range names {
		sum, err := sha256File(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		// Two-space separator matches `sha256sum -c` output.
		fmt.Fprintf(&sb, "%s  %s\n", sum, name)
	}
	if err := os.WriteFile(filepath.Join(dir, "SHA256SUMS"), []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("write SHA256SUMS: %w", err)
	}
	return nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// changelogSection extracts the "## v<version>" (or "## <version>") section of
// a Keep-a-Changelog file, up to (not including) the next section header.
// A missing section yields an empty string without error.
func changelogSection(changelogPath, version string) (string, error) {
	version = strings.TrimPrefix(version, "v")
	data, err := os.ReadFile(changelogPath)
	if err != nil {
		return "", fmt.Errorf("read changelog: %w", err)
	}
	hdr := regexp.MustCompile(`^##\s+v?` + regexp.QuoteMeta(version) + `\s*$`)
	lines := strings.Split(string(data), "\n")

	start := -1
	for i, line := range lines {
		if hdr.MatchString(line) {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return "", nil
	}

	var out []string
	for _, line := range lines[start:] {
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "##\t") {
			break
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n")), nil
}
