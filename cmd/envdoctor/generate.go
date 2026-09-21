package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/haneefojay/envdoctor/internal/contract"
	"github.com/haneefojay/envdoctor/internal/diagnostic"
	"github.com/haneefojay/envdoctor/internal/generate"
	"github.com/haneefojay/envdoctor/internal/output"
)

// defaultExampleFile is the output filename envdoctor generate example writes
// when --output is not supplied.
const defaultExampleFile = ".env.example"

const generateUsage = `Usage: envdoctor generate example [flags]

Generate the .env.example file from the configuration contract.

Only an explicit non-secret default becomes a generated value; secrets and
variables without a default are emitted blank. Existing files are never
silently overwritten.

Flags:
  --contract <path>   Contract file (default "` + defaultContractFile + `")
  --output <path>     Output file (default "` + defaultExampleFile + `")
  -h, --help          Show this help

Exit codes:
  0  file generated
  2  usage, contract, or file error
`

// runGenerate dispatches to the generate example subcommand. "example" is the
// only MVP subcommand; anything else is a usage error.
func runGenerate(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, generateUsage)
		return 2
	}
	if args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(stdout, generateUsage)
		return 0
	}
	if args[0] != "example" {
		fmt.Fprintf(stderr, "envdoctor generate: unknown subcommand %q; expected \"example\"\n\n", args[0])
		fmt.Fprint(stderr, generateUsage)
		return 2
	}
	return runGenerateExample(args[1:], stdout, stderr)
}

func runGenerateExample(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("generate example", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {}

	var contractPath, outputPath string
	fs.StringVar(&contractPath, "contract", "", "contract file path")
	fs.StringVar(&outputPath, "output", "", "output file path")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, generateUsage)
			return 0
		}
		fmt.Fprintf(stderr, "envdoctor generate example: %v\n\n", err)
		fmt.Fprint(stderr, generateUsage)
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "envdoctor generate example: unexpected argument %q\n\n", fs.Arg(0))
		fmt.Fprint(stderr, generateUsage)
		return 2
	}
	if contractPath == "" {
		contractPath = defaultContractFile
	}
	if outputPath == "" {
		outputPath = defaultExampleFile
	}

	data, err := os.ReadFile(contractPath)
	if err != nil {
		return failGenerate(stdout, diagnostic.Errorf(
			diagnostic.CodeContractInvalid, "",
			"Cannot read contract file %q: %v", contractPath, err))
	}

	c, diags := contract.Parse(data, contractPath)
	if c == nil {
		return failGenerate(stdout, diags...)
	}

	content, err := generate.Example(c)
	if err != nil {
		// A parsed contract that cannot be expressed as .env.example is a
		// contract problem, not an internal failure.
		return failGenerate(stdout, diagnostic.Errorf(
			diagnostic.CodeContractInvalid, "",
			"Cannot generate %q from contract: %v", outputPath, err))
	}

	if err := generate.WriteExample(outputPath, content); err != nil {
		if os.IsExist(err) {
			fmt.Fprintf(stderr, "envdoctor generate example: %s already exists; not overwriting\n", outputPath)
		} else {
			fmt.Fprintf(stderr, "envdoctor generate example: %v\n", err)
		}
		return 2
	}

	fmt.Fprintf(stdout, "Wrote %s\n", outputPath)
	return 0
}

// failGenerate renders pre-write contract errors and returns the
// contract/source exit code (2).
func failGenerate(stdout io.Writer, diags ...diagnostic.Diagnostic) int {
	if err := output.WriteHuman(stdout, output.NewResult(diags)); err != nil {
		return 3
	}
	return 2
}
