package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// decodeObjectFromRaw decodes a raw JSON value as a single JSON object with
// duplicate-key detection.
func decodeObjectFromRaw(raw json.RawMessage) (map[string]json.RawMessage, error) {
	if !startsWithObject(raw) {
		return nil, fmt.Errorf("must be a JSON object")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	return decodeObject(dec)
}

func startsWithObject(raw json.RawMessage) bool {
	for _, b := range raw {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		case '{':
			return true
		default:
			return false
		}
	}
	return false
}

// interpretValue interprets a raw JSON value as the given primitive type,
// producing the normalized Go value for that type: string, int64, float64, or
// bool.
func interpretValue(typ Type, raw json.RawMessage) (Value, error) {
	switch typ {
	case TypeString:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, fmt.Errorf("expected a JSON string")
		}
		return s, nil
	case TypeInteger:
		if !isJSONNumber(raw) {
			return nil, fmt.Errorf("expected a JSON number")
		}
		var n json.Number
		_ = json.Unmarshal(raw, &n)
		i, ok := integralJSONNumber(n)
		if !ok {
			return nil, fmt.Errorf("expected an integer value, got %s", n.String())
		}
		return i, nil
	case TypeNumber:
		if !isJSONNumber(raw) {
			return nil, fmt.Errorf("expected a JSON number")
		}
		var n json.Number
		_ = json.Unmarshal(raw, &n)
		f, err := n.Float64()
		if err != nil {
			return nil, fmt.Errorf("number is not representable as a 64-bit float")
		}
		return f, nil
	case TypeBoolean:
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, fmt.Errorf("expected a JSON boolean")
		}
		return b, nil
	default:
		return nil, fmt.Errorf("unsupported type %q", typ)
	}
}

// isJSONNumber reports whether the raw value is a JSON number literal. A naive
// json.Unmarshal into json.Number also accepts JSON strings such as "5", which
// must be rejected: only actual number tokens are numbers here.
func isJSONNumber(raw json.RawMessage) bool {
	first := byte(0)
	for _, b := range raw {
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			continue
		}
		first = b
		break
	}
	if first == 0 {
		return false
	}
	if first != '-' && (first < '0' || first > '9') {
		return false
	}
	var n json.Number
	return json.Unmarshal(raw, &n) == nil
}

// integralJSONNumber converts a JSON number to an int64 when it represents a
// mathematical integer within int64 range. Plain integer literals are parsed
// with exact strconv semantics so overflow and underflow are never silently
// rounded into range. Exponent and fractional forms fall back to float
// arithmetic, which the project's value semantics already accept for JSON
// contract values.
func integralJSONNumber(n json.Number) (int64, bool) {
	s := n.String()
	if !strings.ContainsAny(s, ".eE") {
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i, true
		}
		return 0, false
	}
	f, err := n.Float64()
	if err != nil {
		return 0, false
	}
	if math.Trunc(f) != f {
		return 0, false
	}
	if f < -9223372036854775808.0 || f >= 9223372036854775808.0 {
		return 0, false
	}
	return int64(f), true
}

// intMaxValue is the largest int representable on the host platform.
const intMaxValue = int64(^uint(0) >> 1)

// decodeObjectFields decodes a raw JSON value into an object key map with
// duplicate-key detection.
func decodeObjectFields(raw json.RawMessage) (map[string]json.RawMessage, error) {
	return decodeObjectFromRaw(raw)
}
