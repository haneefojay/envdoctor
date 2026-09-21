package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/haneefojay/envdoctor/internal/contract"
	"github.com/haneefojay/envdoctor/internal/diagnostic"
	"github.com/haneefojay/envdoctor/internal/output"
	"github.com/haneefojay/envdoctor/internal/source"
	"github.com/haneefojay/envdoctor/internal/validate"
)

// defaultContractFile is discovered in the working directory when --contract is
// not supplied. Contract discovery is intentionally limited to this single
// documented location; EnvDoctor does not walk parent directories.
const defaultContractFile = "envdoctor.schema.json"

const checkUsage = `Usage: envdoctor check [flags]

Validate one environment source against the configuration contract.

Flags:
  --contract <path>   Contract file (default "` + defaultContractFile + `")
  --env-file <path>   Validate dotenv file <path> (explicit source)
  --environment       Validate the process environment (default)
  --json              Emit machine-readable JSON instead of text
  -h, --help          Show this help

Sources are never merged: select at most one of --env-file and --environment.
When neither is given, the process environment is validated.

Exit codes:
  0  environment valid
  1  validation failure
  2  usage, contract, or source error
  3  unexpected internal error
`

func runCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {}

	var (
		contractPath string
		envFile      string
		useEnv       bool
		jsonOut      bool
	)
	fs.StringVar(&contractPath, "contract", "", "contract file path")
	fs.StringVar(&envFile, "env-file", "", "dotenv file path")
	fs.BoolVar(&useEnv, "environment", false, "validate the process environment")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, checkUsage)
			return 0
		}
		fmt.Fprintf(stderr, "envdoctor check: %v\n\n", err)
		fmt.Fprint(stderr, checkUsage)
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "envdoctor check: unexpected argument %q\n\n", fs.Arg(0))
		fmt.Fprint(stderr, checkUsage)
		return 2
	}
	if envFile != "" && useEnv {
		fmt.Fprintln(stderr, "envdoctor check: --env-file and --environment select different sources; choose one")
		return 2
	}
	if contractPath == "" {
		contractPath = defaultContractFile
	}

	data, err := os.ReadFile(contractPath)
	if err != nil {
		return failCheck(stdout, jsonOut, diagnostic.Errorf(
			diagnostic.CodeContractInvalid, "",
			"Cannot read contract file %q: %v", contractPath, err))
	}

	c, contractDiags := contract.Parse(data, contractPath)
	if c == nil {
		return failCheck(stdout, jsonOut, contractDiags...)
	}

	var env source.Environment
	if envFile != "" {
		env, err = source.LoadDotenv(envFile)
		if err != nil {
			return failCheck(stdout, jsonOut, sourceDiagnostic(envFile, err))
		}
	} else {
		env = source.Process()
	}

	result := output.NewResult(validate.Validate(c, env))
	if err := writeResult(stdout, jsonOut, result); err != nil {
		return 3
	}
	if result.Valid() {
		return 0
	}
	return 1
}

// sourceDiagnostic maps an environment-source load failure to a stable
// diagnostic. Read failures are SOURCE_INVALID; malformed content is
// ENV_FILE_INVALID. Neither message includes variable values.
func sourceDiagnostic(path string, err error) diagnostic.Diagnostic {
	if source.IsParseError(err) {
		return diagnostic.Errorf(
			diagnostic.CodeEnvFileInvalid, "",
			"Invalid environment file %q: %v", path, err)
	}
	return diagnostic.Errorf(
		diagnostic.CodeSourceInvalid, "",
		"Cannot read environment file %q: %v", path, err)
}

// failCheck renders pre-validation contract/source errors and returns the
// contract/source exit code (2).
func failCheck(stdout io.Writer, jsonOut bool, diags ...diagnostic.Diagnostic) int {
	if err := writeResult(stdout, jsonOut, output.NewResult(diags)); err != nil {
		return 3
	}
	return 2
}

func writeResult(stdout io.Writer, jsonOut bool, result output.Result) error {
	if jsonOut {
		return output.WriteJSON(stdout, result)
	}
	return output.WriteHuman(stdout, result)
}
