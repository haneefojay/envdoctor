package main

import (
	"bytes"
	"strings"
	"testing"
)

func runCLI(args ...string) (code int, stdout, stderr string) {
	var out, err bytes.Buffer
	code = run(args, &out, &err)
	return code, out.String(), err.String()
}

func TestReleaseCLIHelp(t *testing.T) {
	for _, arg := range []string{"-h", "--help"} {
		code, stdout, stderr := runCLI(arg)
		if code != 0 || !strings.Contains(stdout, "Usage: release") || stderr != "" {
			t.Errorf("%s: exit=%d stdout=%q stderr=%q", arg, code, stdout, stderr)
		}
	}
}

func TestReleaseCLIUsageErrors(t *testing.T) {
	cases := [][]string{
		{},                              // missing -version
		{"-version", "not-a-version"},   // invalid semver
		{"-version", "v0.1.0", "extra"}, // unexpected arg
		{"-bogus"},                      // unknown flag
	}
	for _, args := range cases {
		code, _, _ := runCLI(args...)
		if code != 2 {
			t.Errorf("runCLI(%v) exit = %d, want 2", args, code)
		}
	}
}
