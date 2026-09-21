package contract

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// jsonSchema2020_12 is the only accepted $schema value (contract version gate).
const jsonSchema2020_12 = "https://json-schema.org/draft/2020-12/schema"

// Parse decodes the contract JSON into a raw object tree with duplicate-key
// detection, then normalizes it into a Contract. It returns diagnostics in
// deterministic order. The returned Contract is nil when any error-severity
// diagnostic is present.
//
// docPath is used only for error messages and must never contain secrets.
func Parse(data []byte, docPath string) (*Contract, []Diagnostic) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	keys, err := decodeObject(dec)
	if err != nil {
		var dup *duplicateKeyError
		if errors.As(err, &dup) {
			return nil, diags(diagContractInvalid("", "Contract contains duplicate key %q.", dup.key))
		}
		return nil, diags(diagContractInvalid("", "Contract is not valid JSON: %v.", err))
	}
	// Ensure the JSON document is exactly one object value.
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return nil, diags(diagContractInvalid("", "Contract contains trailing content after the root object."))
		}
		return nil, diags(diagContractInvalid("", "Contract is not valid JSON: %v.", err))
	}

	return normalize(keys, docPath)
}

// duplicateKeyError reports a duplicate JSON object key.
type duplicateKeyError struct{ key string }

func (e *duplicateKeyError) Error() string {
	return fmt.Sprintf("duplicate key %q", e.key)
}

// decodeObject consumes a single JSON object starting at the current decoder
// position, returning its keys mapped to raw values. Duplicate keys are
// rejected. The decoder must be positioned at '{'.
func decodeObject(dec *json.Decoder) (map[string]json.RawMessage, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '{' {
		return nil, errors.New("contract root must be a JSON object")
	}

	out := make(map[string]json.RawMessage, 8)
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, errors.New("object key is not a string")
		}
		if _, exists := out[key]; exists {
			return nil, &duplicateKeyError{key: key}
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
		out[key] = raw
	}
	if _, err := dec.Token(); err != nil { // consume closing '}'
		return nil, err
	}
	return out, nil
}
