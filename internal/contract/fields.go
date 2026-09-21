package contract

import (
	"encoding/json"
	"errors"
)

// decodeTypeField decodes a type keyword restricted to the allowed set.
// present reports whether the field exists; ok reports whether it decoded to an
// allowed type.
func decodeTypeField(m map[string]json.RawMessage, key, variable string, allowed map[Type]bool) (Type, bool, bool, []Diagnostic) {
	raw, present := m[key]
	if !present {
		return "", false, false, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", true, false, diags(diagContractInvalid(variable, "Keyword %q must be a JSON string.", key))
	}
	t := Type(s)
	if !allowed[t] {
		return "", true, false, diags(diagContractInvalid(variable, "Unsupported %q %q.", "type", s))
	}
	return t, true, true, nil
}

// decodeStringField decodes a JSON-string keyword. An absent field returns "".
func decodeStringField(m map[string]json.RawMessage, key string) (string, []Diagnostic) {
	raw, present := m[key]
	if !present {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", diags(diagContractInvalid("", "Keyword %q must be a JSON string.", key))
	}
	return s, nil
}

// decodeRequiredField decodes the root required array.
func decodeRequiredField(m map[string]json.RawMessage) ([]string, []Diagnostic) {
	raw, present := m["required"]
	if !present {
		return nil, nil
	}
	items, err := decodeStringArray(raw)
	if err != nil {
		return nil, diags(diagContractInvalid("", "%v.", err))
	}
	seen := make(map[string]bool, len(items))
	for _, name := range items {
		if seen[name] {
			return nil, diags(diagContractInvalid(name, "Variable %q is listed more than once in %q.", name, "required"))
		}
		seen[name] = true
	}
	return items, nil
}

// decodeStringArray decodes a JSON array of strings.
func decodeStringArray(raw json.RawMessage) ([]string, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, errors.New("Keyword must be a JSON array of strings")
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		var s string
		if err := json.Unmarshal(it, &s); err != nil {
			return nil, errors.New("Keyword must be a JSON array of strings")
		}
		out = append(out, s)
	}
	return out, nil
}

// decodePropertiesMap decodes the root properties object, reporting duplicate
// variable definitions as contract duplicate-variable diagnostics.
func decodePropertiesMap(m map[string]json.RawMessage) (map[string]json.RawMessage, []Diagnostic) {
	raw, present := m["properties"]
	if !present {
		return nil, nil
	}
	props, err := decodeObjectFromRaw(raw)
	if err != nil {
		var dup *duplicateKeyError
		if errors.As(err, &dup) {
			return nil, diags(diagDuplicateVariable(dup.key, "Variable %q is defined more than once.", dup.key))
		}
		return nil, diags(diagContractInvalid("", "Keyword %q must be a JSON object: %v.", "properties", err))
	}
	return props, nil
}

// decodeSecretField decodes x-envdoctor-secret (must be a JSON boolean).
func decodeSecretField(m map[string]json.RawMessage) (bool, []Diagnostic) {
	raw, present := m["x-envdoctor-secret"]
	if !present {
		return false, nil
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err != nil {
		return false, diags(diagContractInvalid("", "Keyword %q must be a JSON boolean.", "x-envdoctor-secret"))
	}
	return b, nil
}

// decodeDefaultField interprets a default value for the declared type.
func decodeDefaultField(m map[string]json.RawMessage, typ Type) (Value, []Diagnostic) {
	return decodeTypedValue(m, "default", typ)
}

// decodeConstField interprets a const value for the declared type.
func decodeConstField(m map[string]json.RawMessage, typ Type) (Value, []Diagnostic) {
	return decodeTypedValue(m, "const", typ)
}

func decodeTypedValue(m map[string]json.RawMessage, key string, typ Type) (Value, []Diagnostic) {
	raw, present := m[key]
	if !present {
		return nil, nil
	}
	val, err := interpretValue(typ, raw)
	if err != nil {
		return nil, diags(diagContractInvalid("", "Keyword %q is not a valid %s value.", key, typ))
	}
	return val, nil
}

// decodeEnumField interprets an enum array for the declared type.
func decodeEnumField(m map[string]json.RawMessage, typ Type) ([]Value, []Diagnostic) {
	raw, present := m["enum"]
	if !present {
		return nil, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, diags(diagContractInvalid("", "Keyword %q must be a JSON array.", "enum"))
	}
	if len(items) == 0 {
		return nil, diags(diagContractInvalid("", "Keyword %q must not be an empty array.", "enum"))
	}
	vals := make([]Value, 0, len(items))
	var out []Diagnostic
	for i, it := range items {
		val, err := interpretValue(typ, it)
		if err != nil {
			out = append(out, diagContractInvalid("", "Enum entry %d for type %s is invalid: %v.", i+1, typ, err))
			continue
		}
		vals = append(vals, val)
	}
	return vals, out
}

// decodePatternField decodes the pattern keyword as a JSON string.
func decodePatternField(m map[string]json.RawMessage) (string, []Diagnostic) {
	return decodeStringField(m, "pattern")
}

// decodeIntField decodes a non-negative integer keyword (minLength/maxLength).
func decodeIntField(m map[string]json.RawMessage, key, variable string) (*int, []Diagnostic) {
	raw, present := m[key]
	if !present {
		return nil, nil
	}
	if !isJSONNumber(raw) {
		return nil, diags(diagContractInvalid(variable, "Keyword %q must be a non-negative integer in variable %q.", key, variable))
	}
	var num json.Number
	_ = json.Unmarshal(raw, &num)
	i, ok := integralJSONNumber(num)
	if !ok || i < 0 {
		return nil, diags(diagContractInvalid(variable, "Keyword %q must be a non-negative integer in variable %q.", key, variable))
	}
	if i > intMaxValue {
		return nil, diags(diagContractInvalid(variable, "Keyword %q is too large in variable %q.", key, variable))
	}
	n := int(i)
	return &n, nil
}

// decodeBoundField decodes a numeric bound keyword (minimum etc.).
func decodeBoundField(m map[string]json.RawMessage, key, variable string) (*float64, []Diagnostic) {
	raw, present := m[key]
	if !present {
		return nil, nil
	}
	if !isJSONNumber(raw) {
		return nil, diags(diagContractInvalid(variable, "Keyword %q must be a number in variable %q.", key, variable))
	}
	var num json.Number
	if err := json.Unmarshal(raw, &num); err != nil {
		return nil, diags(diagContractInvalid(variable, "Keyword %q must be a number in variable %q.", key, variable))
	}
	f, err := num.Float64()
	if err != nil {
		return nil, diags(diagContractInvalid(variable, "Keyword %q is not representable in variable %q.", key, variable))
	}
	return &f, nil
}

// decodePositiveNumberField decodes a strictly positive number keyword (multipleOf).
func decodePositiveNumberField(m map[string]json.RawMessage, key, variable string) (*float64, []Diagnostic) {
	f, errDiags := decodeBoundField(m, key, variable)
	if errDiags != nil || f == nil {
		return f, errDiags
	}
	if *f <= 0 {
		return nil, diags(diagContractInvalid(variable, "Keyword %q must be greater than zero in variable %q.", key, variable))
	}
	return f, nil
}

// decodeFormatField validates the format keyword against the supported set.
func decodeFormatField(m map[string]json.RawMessage, variable string) (string, []Diagnostic) {
	if _, present := m["format"]; !present {
		return "", nil
	}
	s, errDiags := decodeStringField(m, "format")
	if errDiags != nil {
		return "", errDiags
	}
	if s != FormatEmail && s != FormatURI {
		return "", diags(diagContractInvalid(variable, "Unsupported format %q in variable %q.", s, variable))
	}
	return s, nil
}
