package contract

import (
	"encoding/json"
	"math"
	"testing"
)

func jsonNumber(s string) json.Number { return json.Number(s) }

func ptr[T any](v T) *T { return &v }

func TestInNumericRangeIntegerBounds(t *testing.T) {
	v := &Variable{Type: TypeInteger}
	assertRange(t, v, int64(5), true)

	v.Minimum = ptr(1.0)
	v.Maximum = ptr(10.0)
	assertRange(t, v, int64(1), true)
	assertRange(t, v, int64(10), true)
	assertRange(t, v, int64(0), false)
	assertRange(t, v, int64(11), false)
}

func TestInNumericRangeExclusiveBounds(t *testing.T) {
	v := &Variable{Type: TypeInteger, ExclusiveMinimum: ptr(1.0), ExclusiveMaximum: ptr(10.0)}
	assertRange(t, v, int64(2), true)
	assertRange(t, v, int64(1), false) // exclusive minimum excludes the bound itself
	assertRange(t, v, int64(10), false)
}

func TestInNumericRangeFractionalBoundsInt64(t *testing.T) {
	// Integer values vs fractional inclusive bounds.
	v := &Variable{Type: TypeInteger, Minimum: ptr(1.5), Maximum: ptr(5.5)}
	assertRange(t, v, int64(2), true)
	assertRange(t, v, int64(5), true)
	assertRange(t, v, int64(1), false)
}

func TestInNumericRangeFloatBounds(t *testing.T) {
	v := &Variable{Type: TypeNumber, Minimum: ptr(0.0), Maximum: ptr(1.0)}
	assertRange(t, v, 0.5, true)
	assertRange(t, v, 1.0, true)
	assertRange(t, v, -0.1, false)

	v = &Variable{Type: TypeNumber, ExclusiveMaximum: ptr(3.14)}
	assertRange(t, v, 3.14, false)
	assertRange(t, v, 3.13, true)
}

func TestInNumericRangeMultipleOfInteger(t *testing.T) {
	v := &Variable{Type: TypeInteger, MultipleOf: ptr(3.0)}
	for _, n := range []int64{0, 3, 6, 9, -3} {
		assertRange(t, v, n, true)
	}
	for _, n := range []int64{1, 2, 4, 5, 7, 8, 10, -1, -2} {
		assertRange(t, v, n, false)
	}
}

func TestInNumericRangeMultipleOfFloat(t *testing.T) {
	v := &Variable{Type: TypeNumber, MultipleOf: ptr(0.5)}
	assertRange(t, v, 1.5, true)
	assertRange(t, v, 1.25, false)
}

func TestInNumericRangeCombined(t *testing.T) {
	v := &Variable{
		Type:             TypeInteger,
		Minimum:          ptr(0.0),
		ExclusiveMaximum: ptr(10.0),
		MultipleOf:       ptr(2.0),
	}
	assertRange(t, v, int64(4), true)
	assertRange(t, v, int64(6), true)
	assertRange(t, v, int64(3), false) // not a multiple
	assertRange(t, v, int64(10), false)
}

func TestInNumericRangeNoBoundsAlwaysTrue(t *testing.T) {
	v := &Variable{Type: TypeInteger}
	assertRange(t, v, int64(math.MaxInt64), true)
	assertRange(t, v, int64(math.MinInt64), true)
	v = &Variable{Type: TypeNumber}
	assertRange(t, v, 1e308, true)
}

func assertRange(t *testing.T, v *Variable, val Value, want bool) {
	t.Helper()
	if got := v.InNumericRange(val); got != want {
		t.Errorf("InNumericRange(%v) = %v, want %v (bounds %+v)", val, got, want, v)
	}
}

func TestInLengthRangeRuneCounting(t *testing.T) {
	v := &Variable{Type: TypeString, MinLength: ptr(1), MaxLength: ptr(3)}
	if !v.InLengthRange("ab") {
		t.Errorf("ab should be in range")
	}
	if v.InLengthRange("") {
		t.Errorf("empty should be out of range")
	}
	if v.InLengthRange("abcd") {
		t.Errorf("abcd should be out of range")
	}
	// Length counts runes, not bytes: "é" is one rune, two bytes.
	v = &Variable{Type: TypeString, MinLength: ptr(1), MaxLength: ptr(1)}
	if !v.InLengthRange("é") {
		t.Errorf("é (1 rune) should be in range")
	}
	if v.InLengthRange("日本") {
		t.Errorf("日本 (2 runes) should be out of range")
	}
	v = &Variable{Type: TypeString}
	if !v.InLengthRange("anything") {
		t.Errorf("unconstrained length should accept anything")
	}
}

func TestMultipleOfFractionalInteger(t *testing.T) {
	// integer value with a fractional multipleOf divisor
	if !multipleOf(int64(6), 0.5) {
		t.Errorf("6 should be a multiple of 0.5")
	}
	if multipleOf(int64(7), 0.3) {
		t.Errorf("7 should not be a multiple of 0.3")
	}
}

func TestMultipleOfFloatSemantics(t *testing.T) {
	if !multipleOf(3.0, 1.5) {
		t.Errorf("3 should be a multiple of 1.5")
	}
	if multipleOf(3.2, 1.5) {
		t.Errorf("3.2 should not be a multiple of 1.5")
	}
	if !multipleOf(0.0, 0.5) {
		t.Errorf("zero is a multiple of any positive divisor")
	}
	if multipleOf(3.0, 0.0) {
		t.Errorf("zero divisor must return false")
	}
}

func TestMultipleOfNegativeDivisorSafe(t *testing.T) {
	// The contract rejects multipleOf <= 0, so the helper never sees it in
	// practice; it must still behave deterministically if it does.
	if !multipleOf(int64(4), -2.0) {
		t.Errorf("magnitude-multiples with a negative divisor stay deterministic")
	}
}

func TestValuesEqualMixedTypes(t *testing.T) {
	if valuesEqual(int64(1), 1.0) {
		t.Errorf("int64 must not equal float64")
	}
	if valuesEqual("1", int64(1)) {
		t.Errorf("string must not equal int64")
	}
	if valuesEqual(true, int64(1)) {
		t.Errorf("bool must not equal int64")
	}
	if !valuesEqual(int64(7), int64(7)) {
		t.Errorf("equal int64 should be equal")
	}
	if !valuesEqual("s", "s") {
		t.Errorf("equal strings should be equal")
	}
	if !valuesEqual(true, true) {
		t.Errorf("equal bools should be equal")
	}
	if valuesEqual(int64(7), int64(8)) {
		t.Errorf("different int64 should differ")
	}
}

func TestIntegralJSONNumber(t *testing.T) {
	cases := []struct {
		in   string
		ok   bool
		want int64
	}{
		{"0", true, 0},
		{"1", true, 1},
		{"3000", true, 3000},
		{"-10", true, -10},
		{"1234567890123456789", true, 1234567890123456789},
		{"3.14", false, 0},
		{"-3.14", false, 0},
		{"1e3", true, 1000}, // JSON exponent parses to an integer value
		{"1.0", true, 1},    // integral float
		{"9223372036854775807", true, math.MaxInt64},
		{"9223372036854775808", false, 0}, // overflow
		{"-9223372036854775808", true, math.MinInt64},
		{"-9223372036854775809", false, 0}, // underflow
		{"1e20", false, 0},                 // beyond int64
	}
	for _, tc := range cases {
		got, ok := integralJSONNumber(jsonNumber(tc.in))
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("integralJSONNumber(%s) = (%d,%v), want (%d,%v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestFloatAsInt64(t *testing.T) {
	if got, ok := floatAsInt64(3.0); !ok || got != 3 {
		t.Errorf("floatAsInt64(3) = (%d,%v)", got, ok)
	}
	if got, ok := floatAsInt64(-3.0); !ok || got != -3 {
		t.Errorf("floatAsInt64(-3) = (%d,%v)", got, ok)
	}
	if _, ok := floatAsInt64(3.5); ok {
		t.Errorf("floatAsInt64(3.5) should fail")
	}
	if _, ok := floatAsInt64(math.Exp2(63)); ok {
		t.Errorf("floatAsInt64(2^63) should fail")
	}
}

func TestNilReceiverSafety(t *testing.T) {
	var c *Contract
	if c.Variable("x") != nil {
		t.Errorf("nil Contract.Variable must return nil")
	}
	if c.VariableNames() != nil {
		t.Errorf("nil Contract.VariableNames must return nil")
	}
}
