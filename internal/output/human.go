package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/haneefojay/envdoctor/internal/diagnostic"
)

// WriteHuman writes the terminal-friendly report for r. The output is
// deterministic, contains no ANSI escape sequences, and never includes
// environment values.
func WriteHuman(w io.Writer, r Result) error {
	var b strings.Builder

	errors, warnings := r.counts()
	if len(r.diagnostics) == 0 {
		b.WriteString("Environment is valid\n")
	} else {
		if r.Valid() {
			b.WriteString("Environment is valid, with warnings\n")
		} else {
			b.WriteString("Environment validation failed\n")
		}
		b.WriteString("\n")
		for i, d := range r.diagnostics {
			if i > 0 {
				b.WriteString("\n")
			}
			writeHumanDiagnostic(&b, d)
		}
		b.WriteString("\n")
		b.WriteString(summaryLine(errors, warnings))
		b.WriteString("\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}

// writeHumanDiagnostic writes one diagnostic block:
//
//	<symbol> <subject>
//	  <SEVERITY> <code>
//	  <message>
//
// subject is the variable name when present, otherwise the message itself, for
// contract- and source-level findings that name no variable. The message is
// printed as its own line only when it is not already the subject.
func writeHumanDiagnostic(b *strings.Builder, d diagnostic.Diagnostic) {
	subject := d.Variable
	if subject == "" {
		subject = d.Message
	}
	fmt.Fprintf(b, "%s %s\n", severitySymbol(d.Severity), subject)
	fmt.Fprintf(b, "  %s %s\n", severityLabel(d.Severity), d.Code)
	if d.Variable != "" {
		fmt.Fprintf(b, "  %s\n", d.Message)
	}
}

// severitySymbol is the marker shown at the start of a diagnostic block.
func severitySymbol(s diagnostic.Severity) string {
	if s == diagnostic.SeverityWarning {
		return "⚠"
	}
	return "✗"
}

// severityLabel is the short uppercase severity shown beside the code.
func severityLabel(s diagnostic.Severity) string {
	if s == diagnostic.SeverityWarning {
		return "WARN"
	}
	return "ERROR"
}

// summaryLine renders the trailing counts, for example "2 errors, 1 warning".
func summaryLine(errors, warnings int) string {
	return fmt.Sprintf("%s, %s", pluralized(errors, "error"), pluralized(warnings, "warning"))
}

// pluralized renders a count and its noun with correct pluralization.
func pluralized(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
