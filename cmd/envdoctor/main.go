package main

import (
	"fmt"
	"io"
	"os"
)

// version is the semantic version of this build. The default is the latest
// released version so that `go install ...@latest` reports a truthful version
// even though it cannot pass -ldflags; tagged releases compiled via the
// release tool override it with `-X
// github.com/haneefojay/envdoctor/cmd/envdoctor.version=<vX.Y.Z>`. Keep the
// default in sync with the latest release (see docs/release.md).
var version = "0.1.0"

const usage = `EnvDoctor - configuration contract validation

Usage:
  envdoctor <command> [flags]

Commands:
  init             Create an initial EnvDoctor contract
  check            Validate an environment against the contract
  generate         Generate derived files (e.g. .env.example)
  version          Print version information
  help             Show this help

Run "envdoctor <command> --help" for command-specific help.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes one EnvDoctor invocation and returns the process exit code.
// Output is written to stdout and stderr so that callers (including tests) can
// capture it. run never reads from stdin and never prompts.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}

	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "envdoctor %s\n", version)
		return 0
	case "help", "--help", "-h":
		fmt.Fprint(stdout, usage)
		return 0
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "generate":
		return runGenerate(args[1:], stdout, stderr)
	case "init":
		return runInit(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "envdoctor: unknown command %q\n\n", args[0])
		fmt.Fprint(stderr, usage)
		return 2
	}
}
