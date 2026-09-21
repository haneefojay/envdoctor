package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/haneefojay/envdoctor/internal/contract"
	"github.com/haneefojay/envdoctor/internal/diagnostic"
	"github.com/haneefojay/envdoctor/internal/source"
	"github.com/haneefojay/envdoctor/internal/validate"
)

const repoContractName = "envdoctor.schema.json"

// loadRepoContract loads EnvDoctor's own repository contract from the
// repository root and asserts it parses cleanly. The repository's contract is
// product data: it must always stay valid under the EnvDoctor Contract
// Profile.
func loadRepoContract(t *testing.T) *contract.Contract {
	t.Helper()
	path := filepath.Join(repoRoot(t), repoContractName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read repo contract: %v", err)
	}
	c, diags := contract.Parse(data, path)
	if c == nil {
		t.Fatalf("repo contract did not parse: %+v", diags)
	}
	return c
}

// TestRepositoryContractValidatesItsOwnCIBuildMatrix validates EnvDoctor's
// own contract against every combination the repository's CI crossbuild job
// actually sets as environment variables, and locks the property that no
// repository variable is required (so a plain `envdoctor check` in the repo
// never fails on an arbitrary developer machine).
func TestRepositoryContractValidatesItsOwnCIBuildMatrix(t *testing.T) {
	c := loadRepoContract(t)

	if got := len(c.VariableNames()); got != 3 {
		t.Errorf("repo contract declares %d variables, want 3", got)
	}
	for _, v := range c.Variables {
		if v.Required {
			t.Errorf("repo contract variable %s is required; repo builds must not depend on ambient environment", v.Name)
		}
	}

	goos := []string{"linux", "windows", "darwin"}
	goarch := []string{"amd64", "arm64"}
	for _, os := range goos {
		for _, arch := range goarch {
			env := source.FromProcess([]string{
				"GOOS=" + os,
				"GOARCH=" + arch,
				"CGO_ENABLED=0",
			})
			if diags := validate.Validate(c, env); len(diags) != 0 {
				t.Errorf("cross-build env GOOS=%s GOARCH=%s CGO_ENABLED=0 produced diagnostics: %+v", os, arch, diags)
			}
		}
	}
}

// TestRepositoryContractRejectsBogusBuildSettings locks the failure modes of
// EnvDoctor's own contract: misspelled targets and non-integer cgo flags must
// surface as usable diagnostics against the repository itself.
func TestRepositoryContractRejectsBogusBuildSettings(t *testing.T) {
	c := loadRepoContract(t)

	cases := []struct {
		name string
		env  string
		want diagnostic.Code
	}{
		{name: "unknown GOOS target", env: "GOOS=fuchsia", want: diagnostic.CodeEnvInvalidEnum},
		{name: "unknown GOARCH target", env: "GOARCH=x86_64", want: diagnostic.CodeEnvInvalidEnum},
		{name: "cgo flag is not an integer", env: "CGO_ENABLED=yes", want: diagnostic.CodeEnvTypeMismatch},
		{name: "cgo flag outside enum", env: "CGO_ENABLED=2", want: diagnostic.CodeEnvInvalidEnum},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			variable := nameOf(tc.env)
			diags := validate.Validate(c, source.FromProcess([]string{tc.env}))
			if len(diags) != 1 {
				t.Fatalf("got %d diagnostics, want exactly 1: %+v", len(diags), diags)
			}
			if diags[0].Code != tc.want {
				t.Errorf("code = %s, want %s", diags[0].Code, tc.want)
			}
			if diags[0].Variable != variable {
				t.Errorf("diagnostic variable = %q, want %q", diags[0].Variable, variable)
			}
		})
	}
}

func nameOf(kv string) string {
	for i := 0; i < len(kv); i++ {
		if kv[i] == '=' {
			return kv[:i]
		}
	}
	return ""
}
