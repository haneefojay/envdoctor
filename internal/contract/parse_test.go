package contract

import (
	"testing"

	"github.com/haneefojay/envdoctor/internal/diagnostic"
)

// testSchema is the only accepted $schema value.
const testSchema = "https://json-schema.org/draft/2020-12/schema"

// doc wraps a root object body into a JSON document.
func doc(body string) string { return "{" + body + "}" }

// header returns the always-valid root prefix.
func header() string { return `"$schema":"` + testSchema + `","type":"object"` }

// rootDoc builds a valid root document with an optional properties object.
func rootDoc(properties string) string {
	if properties == "" {
		return doc(header())
	}
	return doc(header() + `,"properties":{` + properties + `}`)
}

// mustParse parses a valid document, failing the test on any error diagnostic.
func mustParse(t *testing.T, jsonStr string) *Contract {
	t.Helper()
	c, ds := Parse([]byte(jsonStr), "test.json")
	if c == nil {
		t.Fatalf("expected valid contract, got diagnostics: %v", ds)
	}
	return c
}

// parseDiags parses a document and returns its diagnostics (for negative tests).
func parseDiags(t *testing.T, jsonStr string) []Diagnostic {
	t.Helper()
	_, ds := Parse([]byte(jsonStr), "test.json")
	return ds
}

// requireCode asserts a diagnostic with the given code is present.
func requireCode(t *testing.T, ds []Diagnostic, code diagnostic.Code) {
	t.Helper()
	for _, d := range ds {
		if d.Code == code {
			return
		}
	}
	t.Fatalf("expected diagnostic code %s, got %v", code, codesOf(ds))
}

// referenceCode asserts a diagnostic with the given code is absent.
func rejectCode(t *testing.T, ds []Diagnostic, code diagnostic.Code) {
	t.Helper()
	for _, d := range ds {
		if d.Code == code {
			t.Fatalf("unexpected diagnostic code %s: %v", code, ds)
		}
	}
}

func codesOf(ds []Diagnostic) []diagnostic.Code {
	out := make([]diagnostic.Code, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.Code)
	}
	return out
}

func TestParseValidMinimal(t *testing.T) {
	c := mustParse(t, rootDoc(`
		"NAME":{"type":"string"},
		"PORT":{"type":"integer"},
		"RATE":{"type":"number"},
		"DEBUG":{"type":"boolean"}
	`))
	if c.Schema != testSchema {
		t.Errorf("Schema = %q, want %q", c.Schema, testSchema)
	}
	got := c.VariableNames()
	want := []string{"DEBUG", "NAME", "PORT", "RATE"}
	if len(got) != len(want) {
		t.Fatalf("VariableNames = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("VariableNames = %v, want %v", got, want)
		}
	}
	if c.Variable("PORT").Type != TypeInteger {
		t.Errorf("PORT type = %q, want integer", c.Variable("PORT").Type)
	}
	if c.Variable("DEBUG").Type != TypeBoolean {
		t.Errorf("DEBUG type = %q, want boolean", c.Variable("DEBUG").Type)
	}
	if c.Variable("RATE").Type != TypeNumber {
		t.Errorf("RATE type = %q, want number", c.Variable("RATE").Type)
	}
	if c.Variable("NOPE") != nil {
		t.Errorf("Variable(NOPE) = non-nil, want nil")
	}
}

func TestParseValidFullFeatured(t *testing.T) {
	c := mustParse(t, doc(`
		"$schema":"`+testSchema+`",
		"type":"object",
		"title":"App",
		"description":"Full test",
		"required":["HOST"],
		"properties":{
			"HOST":{"type":"string","minLength":3,"maxLength":64,"pattern":"^[a-z0-9.]+$","format":"uri"},
			"NAME":{"type":"string","default":"app"},
			"SECRET":{"type":"string","x-envdoctor-secret":true},
			"PORT":{"type":"integer","minimum":1,"maximum":65535,"exclusiveMinimum":0,"exclusiveMaximum":70000,"multipleOf":1,"enum":[80,443,3000],"const":80},
			"RATE":{"type":"number","minimum":0.5,"maximum":1.5},
			"ENABLED":{"type":"boolean","enum":[true,false],"default":false}
		}
	`))
	host := c.Variable("HOST")
	if host.Required != true {
		t.Errorf("HOST.Required = %v, want true", host.Required)
	}
	if host.HasDefault() {
		t.Errorf("HOST must not declare a default")
	}
	if c.Variable("NAME").Default != "app" {
		t.Errorf("NAME.Default = %v, want app", c.Variable("NAME").Default)
	}
	if host.MinLength == nil || *host.MinLength != 3 {
		t.Errorf("HOST.MinLength = %v, want 3", host.MinLength)
	}
	if host.MaxLength == nil || *host.MaxLength != 64 {
		t.Errorf("HOST.MaxLength = %v, want 64", host.MaxLength)
	}
	if host.Pattern == nil || host.PatternRaw != "^[a-z0-9.]+$" {
		t.Errorf("HOST.Pattern = %v, PatternRaw = %q", host.Pattern, host.PatternRaw)
	}
	if host.Format != FormatURI {
		t.Errorf("HOST.Format = %q, want uri", host.Format)
	}

	secret := c.Variable("SECRET")
	if !secret.Secret {
		t.Errorf("SECRET.Secret = false, want true")
	}

	port := c.Variable("PORT")
	if port.Minimum == nil || *port.Minimum != 1 {
		t.Errorf("PORT.Minimum = %v, want 1", port.Minimum)
	}
	if port.Maximum == nil || *port.Maximum != 65535 {
		t.Errorf("PORT.Maximum = %v, want 65535", port.Maximum)
	}
	if port.ExclusiveMinimum == nil || *port.ExclusiveMinimum != 0 {
		t.Errorf("PORT.ExclusiveMinimum = %v, want 0", port.ExclusiveMinimum)
	}
	if port.ExclusiveMaximum == nil || *port.ExclusiveMaximum != 70000 {
		t.Errorf("PORT.ExclusiveMaximum = %v, want 70000", port.ExclusiveMaximum)
	}
	if port.MultipleOf == nil || *port.MultipleOf != 1 {
		t.Errorf("PORT.MultipleOf = %v, want 1", port.MultipleOf)
	}
	wantEnum := []Value{int64(80), int64(443), int64(3000)}
	if len(port.Enum) != 3 {
		t.Fatalf("PORT.Enum = %v, want %v", port.Enum, wantEnum)
	}
	for i, e := range wantEnum {
		if port.Enum[i] != e {
			t.Fatalf("PORT.Enum = %v, want %v", port.Enum, wantEnum)
		}
	}
	if port.Const != int64(80) {
		t.Errorf("PORT.Const = %v, want 80", port.Const)
	}

	if c.Title != "App" || c.Description != "Full test" {
		t.Errorf("Title/Description = %q/%q, want %q/%q", c.Title, c.Description, "App", "Full test")
	}
}

func TestParseInvalidJSON(t *testing.T) {
	for _, input := range []string{
		`not json`,
		``,
		`{`,
		`{"$schema":`,
		`{"a":1,}`,
	} {
		ds := parseDiags(t, input)
		requireCode(t, ds, diagnostic.CodeContractInvalid)
		if !hasErrors(ds) {
			t.Errorf("input %q: expected error severity, got %v", input, ds)
		}
	}
}

func TestParseTrailingContent(t *testing.T) {
	for _, input := range []string{
		rootDoc(`"A":{"type":"string"}`) + ` extra`,
		rootDoc(`"A":{"type":"string"}`) + ` {}`,
		rootDoc(`"A":{"type":"string"}`) + `,`,
		rootDoc(`"A":{"type":"string"}`) + `]`,
	} {
		ds := parseDiags(t, input)
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}
}

func TestParseNonObjectRoot(t *testing.T) {
	for _, input := range []string{
		`[1,2,3]`,
		`"hello"`,
		`42`,
		`true`,
		`null`,
	} {
		ds := parseDiags(t, input)
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}
}

func TestParseDuplicateRootKey(t *testing.T) {
	ds := parseDiags(t, doc(`"$schema":"`+testSchema+`","$schema":"`+testSchema+`"`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestParseDuplicateVariable(t *testing.T) {
	ds := parseDiags(t, rootDoc(`"A":{"type":"string"},"A":{"type":"integer"}`))
	requireCode(t, ds, diagnostic.CodeContractDuplicateVariable)
}

func TestParseDuplicateKeyInVariable(t *testing.T) {
	ds := parseDiags(t, rootDoc(`"A":{"type":"string","type":"integer"}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestParseMissingSchema(t *testing.T) {
	ds := parseDiags(t, doc(`"type":"object"`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestParseUnsupportedSchema(t *testing.T) {
	cases := []string{
		`"https://json-schema.org/draft/07/schema"`,
		`"http://json-schema.org/draft-04/schema#"`,
		`"https://json-schema.org/draft/2020-12/schema#x"`,
		`"not-a-schema"`,
	}
	for _, schema := range cases {
		ds := parseDiags(t, doc(`"$schema":`+schema+`,"type":"object"`))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
	}
}

func TestParseSchemaNotString(t *testing.T) {
	ds := parseDiags(t, doc(`"$schema":42,"type":"object"`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestParseMissingRootType(t *testing.T) {
	ds := parseDiags(t, doc(`"$schema":"`+testSchema+`"`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestParseRootTypeNotObject(t *testing.T) {
	for _, typ := range []string{`"string"`, `"array"`, `"number"`} {
		ds := parseDiags(t, doc(`"$schema":"`+testSchema+`","type":`+typ))
		requireCode(t, ds, diagnostic.CodeContractInvalid)
		// A present-but-invalid root type must yield exactly one diagnostic
		// (the specific "Unsupported type" one), not a duplicate generic one.
		if len(ds) != 1 {
			t.Errorf("type %s: expected exactly 1 diagnostic, got %d: %v", typ, len(ds), ds)
		}
	}
}

func TestParseUnsupportedRootKeyword(t *testing.T) {
	ds := parseDiags(t, doc(`"$schema":"`+testSchema+`","type":"object","additionalProperties":false`))
	requireCode(t, ds, diagnostic.CodeContractUnsupportedKeyword)

	ds = parseDiags(t, doc(`"$schema":"`+testSchema+`","type":"object","$defs":{}`))
	requireCode(t, ds, diagnostic.CodeContractUnsupportedKeyword)
}

func TestParseRequiredUndeclared(t *testing.T) {
	ds := parseDiags(t, doc(`"$schema":"`+testSchema+`","type":"object","required":["NOPE"],"properties":{"A":{"type":"string"}}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestParseRequiredDuplicateName(t *testing.T) {
	ds := parseDiags(t, doc(`"$schema":"`+testSchema+`","type":"object","required":["A","A"],"properties":{"A":{"type":"string"}}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestParseRequiredNotStringArray(t *testing.T) {
	ds := parseDiags(t, doc(`"$schema":"`+testSchema+`","type":"object","required":["A",3],"properties":{"A":{"type":"string"}}`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestParsePropertiesNotObject(t *testing.T) {
	ds := parseDiags(t, doc(`"$schema":"`+testSchema+`","type":"object","properties":[{"type":"string"}]`))
	requireCode(t, ds, diagnostic.CodeContractInvalid)
}

func TestParseDeterministicDiagnostics(t *testing.T) {
	// Unordered property keys must produce the same sorted diagnostics every time.
	body := `"$schema":"` + testSchema + `","type":"object","properties":{
		"Z":{"type":"bogus"},"A":{"type":"array"},"M":{"minimum":2},"G":{"default":1,"type":"string"}
	}`
	var prev []Diagnostic
	for i := 0; i < 5; i++ {
		ds := parseDiags(t, doc(body))
		if hasErrors(ds) && len(prev) != 0 && !equalDiagnostics(prev, ds) {
			t.Fatalf("nondeterministic diagnostics: %v vs %v", prev, ds)
		}
		prev = ds
	}
	// Diagnostics must be sorted by variable then code then message.
	requireDiagOrder(t, prev)
}

func requireDiagOrder(t *testing.T, ds []Diagnostic) {
	t.Helper()
	for i := 1; i < len(ds); i++ {
		a, b := ds[i-1], ds[i]
		if a.Variable != b.Variable {
			if a.Variable > b.Variable {
				t.Fatalf("diagnostics not sorted by variable: %v before %v", a.Variable, b.Variable)
			}
			continue
		}
		if a.Code != b.Code {
			if a.Code > b.Code {
				t.Fatalf("diagnostics not sorted by code: %v before %v", a.Code, b.Code)
			}
			continue
		}
		if a.Message > b.Message {
			t.Fatalf("diagnostics not sorted by message: %q before %q", a.Message, b.Message)
		}
	}
}

func equalDiagnostics(a, b []Diagnostic) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
