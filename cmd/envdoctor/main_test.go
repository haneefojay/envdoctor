package main

import (
	"regexp"
	"testing"
)

func TestVersion(t *testing.T) {
	// placeholder demonstrating test infrastructure wired in Phase 0
	if version == "" {
		t.Fatal("version must not be empty")
	}
}

// TestVersionIsSemver locks the version contract used by release engineering:
// the embedded version (default or -ldflags-injected) must always be a valid
// semantic version.
func TestVersionIsSemver(t *testing.T) {
	re := regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)
	if !re.MatchString(version) {
		t.Fatalf("version %q is not a semantic version (MAJOR.MINOR.PATCH)", version)
	}
}
