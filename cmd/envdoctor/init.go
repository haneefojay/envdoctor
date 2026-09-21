package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	envinit "github.com/haneefojay/envdoctor/internal/init"
)

const initUsage = `envdoctor init [flags]

Create an initial EnvDoctor contract (envdoctor.schema.json) from variable
names discovered in repository configuration.

Variable names are discovered from .env.example (preferred) or .env in the
current directory. init is an adoption assistant, not an oracle: it copies no
values, classifies no secret, and infers no authoritative type, requiredness,
or default. Every discovered variable is emitted as an optional string for
review.

Flags:
  --output <path>   Contract file to write (default "envdoctor.schema.json")
  -h, --help        Show this help

Exit codes:
  0  contract written
  2  usage, discovery, or file error

Discovery precedence: .env.example, then .env. A malformed discovery source is
an error; it is never skipped silently.
`

func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	var outputPath string
	fs.StringVar(&outputPath, "output", defaultContractFile, "contract file to write")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, initUsage)
			return 0
		}
		fmt.Fprintln(stderr, "envdoctor init: "+usageError(err, fs))
		fmt.Fprint(stderr, initUsage)
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "envdoctor init: unexpected argument %q\n\n%s", fs.Arg(0), initUsage)
		return 2
	}

	names, sourceFile, err := envinit.Discover(".")
	if err != nil {
		fmt.Fprintln(stderr, "envdoctor init: "+err.Error())
		return 2
	}
	if len(names) == 0 {
		fmt.Fprintf(stderr, "envdoctor init: %s declared no variable names\n", sourceFile)
		return 2
	}

	content, err := envinit.Template(names)
	if err != nil {
		fmt.Fprintln(stderr, "envdoctor init: "+err.Error())
		return 2
	}

	if err := envinit.WriteSchema(outputPath, content); err != nil {
		if os.IsExist(err) {
			fmt.Fprintf(stderr, "envdoctor init: %s already exists; not overwriting\n", outputPath)
		} else {
			fmt.Fprintln(stderr, "envdoctor init: "+err.Error())
		}
		return 2
	}

	fmt.Fprintf(stdout, "Wrote %s with %d variables discovered from %s.\n", outputPath, len(names), sourceFile)
	fmt.Fprintln(stdout, "Review the draft before running envdoctor check: adjust types, mark secrets, and set required variables.")
	return 0
}

// usageError renders the leading legible portion of a flag.parse error,
// stripping the "flag provided but not defined" verbosity.
func usageError(err error, fs *flag.FlagSet) string {
	return strings.TrimPrefix(err.Error(), "")
}
