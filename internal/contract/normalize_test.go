package contract

import (
	"testing"

	"github.com/haneefojay/envdoctor/internal/diagnostic"
)

func variableDoc(props string) string {
	return rootDoc(props)
}

func TestVariableMissingType(t *testing.T) {
	ds := parseDiags(t, variableDoc(`"A":{"description":"no type"}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestVariableEmptyObject(t *testing.T) {
	ds := parseDiags(t, variableDoc(`"A":{}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestVariableUnsupportedType(t *testing.T) {
	for _, typ := range []string{`"array"`, `"object"`, `"null"`, `"any"`} {
		ds := parseDiags(t, variableDoc(`"A":{"type":`+typ+`}`))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}
}

// TestTypeErrorSingleDiagnostic guards against duplicate overlapping
// diagnostics: an unsupported type and a missing type must each produce
// exactly one CONTRACT_INVALID explaining the problem.
func TestTypeErrorSingleDiagnostic(t *testing.T) {
	cases := []struct {
		name, props, want string
	}{
		{"missing", `"A":{}`, `Variable "A" must declare a valid "type" (one of string, integer, number, boolean).`},
		{"unsupported", `"A":{"type":"array"}`, `Unsupported "type" "array".`},
		{"not-string", `"A":{"type":7}`, `Keyword "type" must be a JSON string.`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ds := parseDiags(t, variableDoc(tc.props))
			if len(ds) != 1 {
				t.Fatalf("expected exactly 1 diagnostic, got %d: %v", len(ds), ds)
			}
			if d := ds[0]; d.Code != diagnostic.CodeContractInvalid || d.Message != tc.want {
				t.Fatalf("unexpected diagnostic: %+v (want %s)", d, tc.want)
			}
		})
	}
}

func TestVariableTypeNotStringValue(t *testing.T) {
	for _, typ := range []string{`7`, `["string"]`, `true`, `{"$ref":"#"}`} {
		ds := parseDiags(t, variableDoc(`"A":{"type":`+typ+`}`))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}
}

func TestVariableUnsupportedKeyword(t *testing.T) {
	cases := []string{
		`"A":{"type":"string","minItems":1}`,
		`"A":{"type":"string","$comment":"x"}`,
		`"A":{"type":"string","items":{"type":"string"}}`,
		`"A":{"type":"string","allOf":[]}`,
	}
	for _, props := range cases {
		ds := parseDiags(t, variableDoc(props))
		requireCode(t, ds, diagnostic.CodeContractUnsupportedKeyword)
	}
}

func TestSecretFlag(t *testing.T) {
	c := mustParse(t, variableDoc(`
		"S1":{"type":"string","x-envdoctor-secret":true},
		"S2":{"type":"string","x-envdoctor-secret":false},
		"S3":{"type":"string"}
	`))
	if !c.Variable("S1").Secret {
		t.Errorf("S1 expected a secret")
	}
	if c.Variable("S2").Secret {
		t.Errorf("S2 not expected to be a secret")
	}
	if c.Variable("S3").Secret {
		t.Errorf("S3 not expected to be a secret")
	}
}

func TestSecretNonBoolean(t *testing.T) {
	for _, v := range []string{`"yes"`, `1`, `"true"`} {
		ds := parseDiags(t, variableDoc(`"A":{"type":"string","x-envdoctor-secret":`+v+`}`))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}
}

func TestSecretDefaultConflict(t *testing.T) {
	ds := parseDiags(t, variableDoc(`"SK":{"type":"string","x-envdoctor-secret":true,"default":"sekret"}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestRequiredDefaultConflict(t *testing.T) {
	ds := parseDiags(t, doc(`"$schema":"`+testSchema+`","type":"object","required":["A"],"properties":{"A":{"type":"string","default":"x"}}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestDefaultTypeMismatch(t *testing.T) {
	cases := []struct {
		name string
		prop string
	}{
		{"string default number", `"A":{"type":"string","default":7}`},
		{"string default bool", `"A":{"type":"string","default":true}`},
		{"integer default string", `"A":{"type":"integer","default":"5"}`},
		{"integer default float", `"A":{"type":"integer","default":5.5}`},
		{"number default string", `"A":{"type":"number","default":"1.5"}`},
		{"boolean default number", `"A":{"type":"boolean","default":1}`},
		{"boolean default string", `"A":{"type":"boolean","default":"true"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ds := parseDiags(t, variableDoc(tc.prop))
			requireCode(t, ds, diagnostic.CodeContractInvalid)
		})
	}
}

func TestEnumInterpretation(t *testing.T) {
	c := mustParse(t, variableDoc(`
		"COLOR":{"type":"string","enum":["red","green","BLUE"]},
		"PORT":{"type":"integer","enum":[80,443,3000]},
		"RATE":{"type":"number","enum":[0.5,1.5,2]},
		"FLAG":{"type":"boolean","enum":[true,false]}
	`))

	got := c.Variable("COLOR").Enum
	want := []Value{"red", "green", "BLUE"}
	assertValues(t, got, want)

	got = c.Variable("PORT").Enum
	assertValues(t, got, []Value{int64(80), int64(443), int64(3000)})

	got = c.Variable("RATE").Enum
	assertValues(t, got, []Value{0.5, 1.5, 2.0})

	got = c.Variable("FLAG").Enum
	assertValues(t, got, []Value{true, false})
}

func assertValues(t *testing.T, got, want []Value) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if !valuesEqual(got[i], want[i]) {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestEnumEmpty(t *testing.T) {
	for _, typ := range []string{`"string"`, `"integer"`, `"number"`, `"boolean"`} {
		ds := parseDiags(t, variableDoc(`"A":{"type":`+typ+`,"enum":[]}`))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}
}

func TestEnumInvalidEntry(t *testing.T) {
	cases := []string{
		`"A":{"type":"integer","enum":[1,"two"]}`,
		`"A":{"type":"integer","enum":[1.5]}`,
		`"A":{"type":"number","enum":["x"]}`,
		`"A":{"type":"boolean","enum":["true"]}`,
		`"A":{"type":"string","enum":[1]}`,
	}
	for _, props := range cases {
		ds := parseDiags(t, variableDoc(props))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}
}

func TestEnumNotArray(t *testing.T) {
	ds := parseDiags(t, variableDoc(`"A":{"type":"string","enum":"red"}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestConstTypeMismatch(t *testing.T) {
	ds := parseDiags(t, variableDoc(`"A":{"type":"integer","const":"80"}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestConstNotInEnum(t *testing.T) {
	ds := parseDiags(t, variableDoc(`"A":{"type":"integer","enum":[1,2],"const":5}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestConstInEnumValid(t *testing.T) {
	c := mustParse(t, variableDoc(`"A":{"type":"string","enum":["x","y"],"const":"x"}`))
	if !c.Variable("A").ConstMatches("x") {
		t.Errorf("const should match x")
	}
}

func TestDefaultNotInEnum(t *testing.T) {
	ds := parseDiags(t, variableDoc(`"A":{"type":"integer","enum":[1,2],"default":3}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestDefaultInEnumValid(t *testing.T) {
	c := mustParse(t, variableDoc(`"A":{"type":"integer","enum":[1,2],"default":2}`))
	if c.Variable("A").Default != int64(2) {
		t.Errorf("Default = %v, want 2", c.Variable("A").Default)
	}
}

func TestDefaultViolatesPattern(t *testing.T) {
	ds := parseDiags(t, variableDoc(`"A":{"type":"string","pattern":"^[0-9]+$","default":"abc"}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestDefaultViolatesLength(t *testing.T) {
	ds := parseDiags(t, variableDoc(`"A":{"type":"string","maxLength":3,"default":"toolong"}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestDefaultViolatesRange(t *testing.T) {
	cases := []string{
		`"A":{"type":"integer","minimum":10,"default":5}`,
		`"A":{"type":"integer","maximum":10,"default":15}`,
		`"A":{"type":"number","exclusiveMinimum":0,"default":0}`,
		`"A":{"type":"integer","multipleOf":3,"default":7}`,
	}
	for _, props := range cases {
		ds := parseDiags(t, variableDoc(props))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}
}

func TestConstViolatesRange(t *testing.T) {
	ds := parseDiags(t, variableDoc(`"A":{"type":"integer","minimum":1,"maximum":10,"const":50}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestDefaultBooleanEnum(t *testing.T) {
	c := mustParse(t, variableDoc(`"A":{"type":"boolean","enum":[false],"default":false}`))
	if c.Variable("A").Default != false {
		t.Errorf("Default = %v, want false", c.Variable("A").Default)
	}

	ds := parseDiags(t, variableDoc(`"A":{"type":"boolean","enum":[false],"default":true}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestPatternValid(t *testing.T) {
	c := mustParse(t, variableDoc(`"A":{"type":"string","pattern":"^[a-zA-Z0-9_]{1,32}$"}`))
	v := c.Variable("A")
	if v.Pattern == nil {
		t.Fatalf("Pattern = nil, want compiled regexp")
	}
	if v.PatternRaw != "^[a-zA-Z0-9_]{1,32}$" {
		t.Errorf("PatternRaw = %q", v.PatternRaw)
	}
}

func TestPatternInvalid(t *testing.T) {
	for _, pat := range []string{`"("`, `"[a-z"`, `"*"`} {
		ds := parseDiags(t, variableDoc(`"A":{"type":"string","pattern":`+pat+`}`))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}
}

func TestLengthConstraintsNormalized(t *testing.T) {
	c := mustParse(t, variableDoc(`"A":{"type":"string","minLength":1,"maxLength":100}`))
	v := c.Variable("A")
	if v.MinLength == nil || *v.MinLength != 1 {
		t.Errorf("MinLength = %v, want 1", v.MinLength)
	}
	if v.MaxLength == nil || *v.MaxLength != 100 {
		t.Errorf("MaxLength = %v, want 100", v.MaxLength)
	}
}

func TestLengthMustBeNonNegativeInteger(t *testing.T) {
	for _, bad := range []string{`-1`, `2.5`, `"3"`, `true`} {
		for _, key := range []string{"minLength", "maxLength"} {
			ds := parseDiags(t, variableDoc(`"A":{"type":"string","`+key+`":`+bad+`}`))
			requireCode(t, ds, diagnostic.CodeContractInvalid)
		}
	}
}

func TestLengthContradiction(t *testing.T) {
	ds := parseDiags(t, variableDoc(`"A":{"type":"string","minLength":5,"maxLength":2}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestNumericBoundsNormalized(t *testing.T) {
	c := mustParse(t, variableDoc(`"A":{"type":"number","minimum":0,"maximum":1,"exclusiveMinimum":-1,"exclusiveMaximum":2,"multipleOf":0.5}`))
	v := c.Variable("A")
	if v.Minimum == nil || *v.Minimum != 0 {
		t.Errorf("Minimum = %v, want 0", v.Minimum)
	}
	if v.Maximum == nil || *v.Maximum != 1 {
		t.Errorf("Maximum = %v, want 1", v.Maximum)
	}
	if v.ExclusiveMinimum == nil || *v.ExclusiveMinimum != -1 {
		t.Errorf("ExclusiveMinimum = %v, want -1", v.ExclusiveMinimum)
	}
	if v.ExclusiveMaximum == nil || *v.ExclusiveMaximum != 2 {
		t.Errorf("ExclusiveMaximum = %v, want 2", v.ExclusiveMaximum)
	}
	if v.MultipleOf == nil || *v.MultipleOf != 0.5 {
		t.Errorf("MultipleOf = %v, want 0.5", v.MultipleOf)
	}
}

func TestNumericBoundsOnIntegerVariables(t *testing.T) {
	c := mustParse(t, variableDoc(`"A":{"type":"integer","minimum":1,"maximum":10}`))
	v := c.Variable("A")
	if v.Minimum == nil || *v.Minimum != 1 {
		t.Errorf("Minimum = %v, want 1", v.Minimum)
	}
}

func TestBoundsNotNumbers(t *testing.T) {
	for _, key := range []string{"minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum"} {
		ds := parseDiags(t, variableDoc(`"A":{"type":"integer","`+key+`":"x"}`))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}
}

func TestMultipleOfNonPositive(t *testing.T) {
	for _, bad := range []string{`0`, `-1`, `0.0`} {
		ds := parseDiags(t, variableDoc(`"A":{"type":"integer","multipleOf":`+bad+`}`))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}
}

func TestNumericContradictions(t *testing.T) {
	cases := []string{
		`"A":{"type":"integer","minimum":10,"maximum":2}`,
		`"A":{"type":"number","exclusiveMinimum":10,"maximum":10}`,
		`"A":{"type":"number","minimum":10,"exclusiveMaximum":10}`,
		`"A":{"type":"number","exclusiveMinimum":10,"exclusiveMaximum":10}`,
	}
	for _, props := range cases {
		ds := parseDiags(t, variableDoc(props))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}
}

func TestCompatibleExclusiveBoundsValid(t *testing.T) {
	// minimum and exclusiveMinimum are both lower bounds and may coexist.
	c := mustParse(t, variableDoc(`"A":{"type":"number","minimum":10,"exclusiveMinimum":5}`))
	if c.Variable("A").Minimum == nil {
		t.Errorf("expected minimum to be retained")
	}
}

func TestFormatNormalized(t *testing.T) {
	c := mustParse(t, variableDoc(`
		"E":{"type":"string","format":"email"},
		"U":{"type":"string","format":"uri"}
	`))
	if c.Variable("E").Format != FormatEmail {
		t.Errorf("E.Format = %q, want email", c.Variable("E").Format)
	}
	if c.Variable("U").Format != FormatURI {
		t.Errorf("U.Format = %q, want uri", c.Variable("U").Format)
	}
}

func TestFormatUnsupported(t *testing.T) {
	for _, f := range []string{`"date-time"`, `"hostname"`, `"ipv4"`, `"custom"`} {
		ds := parseDiags(t, variableDoc(`"A":{"type":"string","format":`+f+`}`))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}
}

func TestFormatNotString(t *testing.T) {
	ds := parseDiags(t, variableDoc(`"A":{"type":"string","format":42}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestNumericConstraintsOnString(t *testing.T) {
	for _, key := range []string{"minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf"} {
		ds := parseDiags(t, variableDoc(`"A":{"type":"string","`+key+`":1}`))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}
}

func TestStringConstraintsOnNonString(t *testing.T) {
	lengthProps := []string{
		`"A":{"type":"integer","minLength":1}`,
		`"A":{"type":"boolean","maxLength":1}`,
	}
	for _, props := range lengthProps {
		ds := parseDiags(t, variableDoc(props))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}

	patternProps := []string{
		`"A":{"type":"integer","pattern":"^[0-9]+$"}`,
		`"A":{"type":"boolean","pattern":"true"}`,
	}
	for _, props := range patternProps {
		ds := parseDiags(t, variableDoc(props))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}

	formatProps := []string{
		`"A":{"type":"integer","format":"uri"}`,
		`"A":{"type":"boolean","format":"email"}`,
	}
	for _, props := range formatProps {
		ds := parseDiags(t, variableDoc(props))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}
}

func TestConstraintsOnBoolean(t *testing.T) {
	for _, props := range []string{
		`"A":{"type":"boolean","minimum":1}`,
		`"A":{"type":"boolean","multipleOf":2}`,
		`"A":{"type":"boolean","maxLength":1}`,
		`"A":{"type":"boolean","pattern":"true"}`,
		`"A":{"type":"boolean","format":"uri"}`,
	} {
		ds := parseDiags(t, variableDoc(props))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}
}

func TestRequiredAppliedFromRoot(t *testing.T) {
	c := mustParse(t, doc(`"$schema":"`+testSchema+`","type":"object","required":["A","B"],"properties":{"A":{"type":"string"},"B":{"type":"string"},"C":{"type":"string"}}`))
	if !c.Variable("A").Required || !c.Variable("B").Required {
		t.Errorf("A and B should be required")
	}
	if c.Variable("C").Required {
		t.Errorf("C should not be required")
	}
}

func TestUnsupportedKeywordBlocksContract(t *testing.T) {
	ds := parseDiags(t, variableDoc(`"A":{"type":"string","unevaluatedProperties":false}`))
	requireCode(t, ds, diagnostic.CodeContractUnsupportedKeyword)
}

func TestParseEmptyPropertiesValid(t *testing.T) {
	c := mustParse(t, doc(`"$schema":"`+testSchema+`","type":"object"`))
	if len(c.Variables) != 0 {
		t.Errorf("Variables = %v, want none", c.Variables)
	}
}
