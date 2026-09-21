package normalize

import (
	"strings"
	"testing"

	"github.com/haneefojay/envdoctor/internal/contract"
)

func TestBooleanAccept(t *testing.T) {
	for _, raw := range []string{"true", "false", "TRUE", "FALSE", "True", "False", "tRuE", "FaLsE"} {
		got, err := Boolean(raw)
		if err != nil {
			t.Errorf("Boolean(%q) failed: %v", raw, err)
			continue
		}
		want := strings.EqualFold(raw, "true")
		if got != want {
			t.Errorf("Boolean(%q) = %v, want %v", raw, got, want)
		}
	}
}

func TestBooleanReject(t *testing.T) {
	for _, raw := range []string{
		"1", "0", "yes", "no", "on", "off", "enabled", "disabled",
		"true ", " true", "tru", "TRUE1", "Truee", "", "y", "n",
		"TRUE\t", "TRUE\n",
	} {
		if _, err := Boolean(raw); err == nil {
			t.Errorf("Boolean(%q) must be rejected", raw)
		}
	}
}

func TestIntegerAccept(t *testing.T) {
	cases := []struct {
		raw  string
		want int64
	}{
		{"0", 0},
		{"1", 1},
		{"3000", 3000},
		{"-10", -10},
		{"+10", 10},
		{"0007", 7},
		{"-0", 0},
		{"+0", 0},
		{"9223372036854775807", 9223372036854775807},
		{"-9223372036854775808", -9223372036854775808},
	}
	for _, tc := range cases {
		got, err := Integer(tc.raw)
		if err != nil {
			t.Errorf("Integer(%q) failed: %v", tc.raw, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Integer(%q) = %d, want %d", tc.raw, got, tc.want)
		}
	}
}

func TestIntegerReject(t *testing.T) {
	for _, raw := range []string{
		"3.14", "1e3", "1E3", "1_000", "0x10", "10ms", "10ms",
		" 5", "5 ", "\t5", "5\t",
		"+", "-", "", ".5", "5.", "--5", "++5", "+-5",
		"1.0", "1,000", "١٢٣", "10,5",
		"9223372036854775808", "-9223372036854775809",
		"true", "NaN", "Infinity", "0b1010",
	} {
		if _, err := Integer(raw); err == nil {
			t.Errorf("Integer(%q) must be rejected", raw)
		}
	}
}

func TestIntegerOutOfRangeDistinct(t *testing.T) {
	if _, err := Integer("9223372036854775808"); err == nil {
		t.Fatalf("overflow must fail")
	}
	if _, err := Integer("-9223372036854775809"); err != ErrIntegerOutOfRange {
		t.Fatalf("expected ErrIntegerOutOfRange, got %v", err)
	}
}

func TestNumberAccept(t *testing.T) {
	cases := []struct {
		raw  string
		want float64
	}{
		{"0", 0},
		{"3", 3},
		{"3.14", 3.14},
		{"-3.14", -3.14},
		{"+3.14", 3.14},
		{".5", 0.5},
		{"-.5", -0.5},
		{"+.5", 0.5},
		{"007", 7},
		{"0.0", 0},
		{"-0.0", 0},
	}
	for _, tc := range cases {
		got, err := Number(tc.raw)
		if err != nil {
			t.Errorf("Number(%q) failed: %v", tc.raw, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Number(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
}

func TestNumberReject(t *testing.T) {
	for _, raw := range []string{
		"1e3", "1E3", "1e+3", "1e-3",
		"1_000", "0x10", "NaN", "nan", "Infinity", "inf", "-Infinity",
		" 3.14", "3.14 ", "\t3", "3\n",
		"", "+", "-", ".", "+.", "-.",
		"a", "3.14.5", "10ms", "1,5", "$5", "--3", "++3", "+-3",
	} {
		if _, err := Number(raw); err == nil {
			t.Errorf("Number(%q) must be rejected", raw)
		}
	}
}

func TestStringPreservesValue(t *testing.T) {
	if got := String("  value  "); got != "  value  " {
		t.Errorf("String must not trim, got %q", got)
	}
	if got := String(""); got != "" {
		t.Errorf("String(\"\") = %q", got)
	}
	for _, raw := range []string{"true", "1", "3.14", "$HOME", "héllo", "日本"} {
		if got, err := Interpret(contract.TypeString, raw); err != nil || got != raw {
			t.Errorf("Interpret(string, %q) = (%v, %v), want (%q, nil)", raw, got, err, raw)
		}
	}
}

func TestInterpretDispatch(t *testing.T) {
	if v, err := Interpret(contract.TypeBoolean, "TRUE"); err != nil || v != true {
		t.Errorf("Interpret(boolean, TRUE) = (%v, %v)", v, err)
	}
	if v, err := Interpret(contract.TypeInteger, "42"); err != nil || v != int64(42) {
		t.Errorf("Interpret(integer, 42) = (%v, %v)", v, err)
	}
	if v, err := Interpret(contract.TypeNumber, "-1.5"); err != nil || v != -1.5 {
		t.Errorf("Interpret(number, -1.5) = (%v, %v)", v, err)
	}
	if _, err := Interpret(contract.Type("weird"), "x"); err != ErrUnsupportedType {
		t.Errorf("Interpret(unsupported) = %v, want ErrUnsupportedType", err)
	}
}

func TestNoTrimmingBeforeParse(t *testing.T) {
	// Values with surrounding whitespace are rejected, not trimmed.
	for _, typ := range []contract.Type{contract.TypeInteger, contract.TypeNumber, contract.TypeBoolean} {
		for _, raw := range []string{" 5", "5 ", " true", "1 "} {
			if _, err := Interpret(typ, raw); err == nil {
				t.Errorf("Interpret(%s, %q) must reject rather than trim", typ, raw)
			}
		}
	}
}
