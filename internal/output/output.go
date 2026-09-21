// Package output renders validation results for humans and machines.
//
// It is a presentation layer: it consumes structured diagnostics and never
// reads the environment, so it cannot disclose environment values that the
// diagnostics do not already carry. Diagnostics are the secret-safety
// boundary, not output.
//
// Rendering is deterministic: the same result always produces byte-identical
// output. Human output emits no ANSI escape sequences, and JSON output is
// stable and machine-readable.
package output

import (
	"sort"

	"github.com/haneefojay/envdoctor/internal/diagnostic"
)

// Result is the deterministic outcome of a validation run.
type Result struct {
	// diagnostics are always kept in canonical order (see sortDiagnostics).
	diagnostics []diagnostic.Diagnostic
}

// NewResult returns a Result whose diagnostics are copied and sorted into the
// canonical deterministic order: error-severity diagnostics before warnings,
// then by variable, code, and message. The input slice is never retained or
// mutated.
func NewResult(diags []diagnostic.Diagnostic) Result {
	sorted := make([]diagnostic.Diagnostic, len(diags))
	copy(sorted, diags)
	sortDiagnostics(sorted)
	return Result{diagnostics: sorted}
}

// Diagnostics returns the diagnostics in canonical order. The returned slice
// is a copy, so mutating it does not affect the Result.
func (r Result) Diagnostics() []diagnostic.Diagnostic {
	out := make([]diagnostic.Diagnostic, len(r.diagnostics))
	copy(out, r.diagnostics)
	return out
}

// Valid reports whether the environment satisfies the contract. It is valid
// unless at least one error-severity diagnostic is present; warnings do not
// invalidate an environment.
func (r Result) Valid() bool {
	for _, d := range r.diagnostics {
		if d.Severity == diagnostic.SeverityError {
			return false
		}
	}
	return true
}

// counts returns the error and warning totals.
func (r Result) counts() (errors, warnings int) {
	for _, d := range r.diagnostics {
		switch d.Severity {
		case diagnostic.SeverityError:
			errors++
		case diagnostic.SeverityWarning:
			warnings++
		}
	}
	return errors, warnings
}

// sortDiagnostics orders diagnostics canonically. The ordering is total, so it
// never depends on the input order or on Go map iteration.
func sortDiagnostics(ds []diagnostic.Diagnostic) {
	sort.SliceStable(ds, func(i, j int) bool {
		a, b := ds[i], ds[j]
		if ra, rb := severityRank(a.Severity), severityRank(b.Severity); ra != rb {
			return ra < rb
		}
		if a.Variable != b.Variable {
			return a.Variable < b.Variable
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Message < b.Message
	})
}

// severityRank orders errors before warnings, then any unknown severity.
func severityRank(s diagnostic.Severity) int {
	switch s {
	case diagnostic.SeverityError:
		return 0
	case diagnostic.SeverityWarning:
		return 1
	default:
		return 2
	}
}
