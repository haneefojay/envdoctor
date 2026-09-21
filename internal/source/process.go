// Package source implements the MVP environment sources and normalizes them to
// the shared representation validator consumes: map[string]string.
//
// Environment variable names are semantically case-sensitive regardless of the
// host operating system. Sources never case-normalize, trim, or otherwise
// rewrite names or values.
package source

import (
	"os"
	"strings"
)

// Environment is the normalized representation of an environment: variable
// names mapped to exact string values. Missing and empty are distinct: an empty
// value is present with its value equal to "".
type Environment = map[string]string

// Process returns the current process environment, preserving names and values
// exactly.
func Process() Environment {
	return FromProcess(os.Environ())
}

// FromProcess normalizes raw "KEY=VALUE" entries into an Environment.
//
// Names and values are preserved verbatim: no case-normalization and no
// trimming. When the same key appears more than once, the last entry wins,
// which keeps behavior deterministic when a platform reports duplicates.
func FromProcess(environ []string) Environment {
	env := make(Environment, len(environ))
	for _, entry := range environ {
		eq := strings.IndexByte(entry, '=')
		if eq < 0 {
			continue
		}
		// Split only at the first '=' so values containing '=' are preserved.
		env[entry[:eq]] = entry[eq+1:]
	}
	return env
}
