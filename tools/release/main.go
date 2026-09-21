// Command release packages EnvDoctor for tagged releases: it builds the CLI
// for every documented target, wraps each binary in a deterministic archive,
// writes a SHA256SUMS manifest, and stages the release notes from the
// CHANGELOG. It is driven by .github/workflows/release.yml.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const usage = `Usage: release [flags]

Build EnvDoctor for every documented target and produce release artifacts.

Run from the repository root. Uses the ` + "`go`" + ` command from PATH; the
module must build with the same Go version used for CI.

Flags:
  -version <vX.Y.Z>   Semantic version to embed and name artifacts (required)
  -out <dir>          Output directory (default "dist")
  -notes <path>       CHANGELOG file; writes <out>/NOTES.md for the version
  -h, --help          Show this help

Artifacts:
  dist/envdoctor_<version>_<os>_<arch>.{tar.gz,zip}   one per documented target
  dist/SHA256SUMS                                      sha256sum-manifest
  dist/NOTES.md                                        changelog section (-notes)

Exit codes:
  0  success
  2  usage or packaging error
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("release", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {}

	var (
		version   string
		outDir    string
		notesPath string
	)
	fs.StringVar(&version, "version", "", "semantic version (required)")
	fs.StringVar(&outDir, "out", "dist", "output directory")
	fs.StringVar(&notesPath, "notes", "", "CHANGELOG file for NOTES.md")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usage)
			return 0
		}
		fmt.Fprint(stderr, usage)
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "release: unexpected argument %q\n\n", fs.Arg(0))
		fmt.Fprint(stderr, usage)
		return 2
	}
	if version == "" {
		fmt.Fprintln(stderr, "release: -version is required")
		fmt.Fprint(stderr, usage)
		return 2
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "release: %v\n", err)
		return 3
	}

	archives, err := Package(root, Options{
		Version: version,
		OutDir:  outDir,
		Notes:   notesPath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "release: %v\n", err)
		return 2
	}

	fmt.Fprintf(stdout, "packaged %d targets for %s\n", len(archives), version)
	for _, archive := range archives {
		fmt.Fprintln(stdout, filepath.Base(archive))
	}
	if notesPath != "" {
		fmt.Fprintln(stdout, filepath.Join(outDir, "NOTES.md"))
	}
	return 0
}
