package output

import (
	"encoding/json"
	"io"

	"github.com/haneefojay/envdoctor/internal/diagnostic"
)

// jsonDocument is the stable machine-readable shape. Field order is fixed by
// struct order, so the encoding is deterministic.
type jsonDocument struct {
	Valid       bool                    `json:"valid"`
	Diagnostics []diagnostic.Diagnostic `json:"diagnostics"`
}

// WriteJSON writes r as the stable machine-readable JSON document
// {"valid": bool, "diagnostics": [...]}. Diagnostics appear in canonical
// order, the diagnostics array is always present, and the output is valid JSON
// with no ANSI escape sequences.
func WriteJSON(w io.Writer, r Result) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(jsonDocument{
		Valid:       r.Valid(),
		Diagnostics: r.Diagnostics(),
	})
}
