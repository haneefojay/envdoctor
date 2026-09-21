package contract

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"

	"github.com/haneefojay/envdoctor/internal/diagnostic"
)

// Diagnostic is the package view of a structured diagnostic.
type Diagnostic = diagnostic.Diagnostic

// diag helper aliases for brevity within the package.
func diagContractInvalid(variable, format string, args ...any) Diagnostic {
	return diagnostic.Errorf(diagnostic.CodeContractInvalid, variable, format, args...)
}

func diagUnsupportedKeyword(variable, format string, args ...any) Diagnostic {
	return diagnostic.Errorf(diagnostic.CodeContractUnsupportedKeyword, variable, format, args...)
}

func diagDuplicateVariable(variable, format string, args ...any) Diagnostic {
	return diagnostic.Errorf(diagnostic.CodeContractDuplicateVariable, variable, format, args...)
}

func diags(ds ...Diagnostic) []Diagnostic { return ds }

// rootKeywords is the set of supported contract-root keywords.
var rootKeywords = map[string]bool{
	"$schema":     true,
	"title":       true,
	"description": true,
	"type":        true,
	"required":    true,
	"properties":  true,
}

// rootTypeField is the set of types accepted for the contract root.
var rootTypeField = map[Type]bool{TypeObject: true}

// variableTypeField is the set of types accepted for an environment variable.
var variableTypeField = map[Type]bool{
	TypeString:  true,
	TypeInteger: true,
	TypeNumber:  true,
	TypeBoolean: true,
}

// variableKeywords is the set of supported per-variable keywords.
var variableKeywords = map[string]bool{
	"type":               true,
	"enum":               true,
	"const":              true,
	"default":            true,
	"description":        true,
	"title":              true,
	"pattern":            true,
	"minLength":          true,
	"maxLength":          true,
	"minimum":            true,
	"maximum":            true,
	"exclusiveMinimum":   true,
	"exclusiveMaximum":   true,
	"multipleOf":         true,
	"format":             true,
	"x-envdoctor-secret": true,
}

// normalize converts the raw contract object tree into a normalized Contract,
// validating the EnvDoctor Contract Profile along the way.
func normalize(root map[string]json.RawMessage, docPath string) (*Contract, []Diagnostic) {
	var out []Diagnostic
	emit := func(d Diagnostic) { out = append(out, d) }

	// Root keyword check.
	for key := range root {
		if !rootKeywords[key] {
			emit(diagUnsupportedKeyword("", "Unsupported contract keyword %q.", key))
		}
	}

	schema, rootDiags := decodeStringField(root, "$schema")
	out = append(out, rootDiags...)
	if schema == "" {
		emit(diagContractInvalid("", "Contract must declare %q with a supported JSON Schema version.", "$schema"))
	} else if schema != jsonSchema2020_12 {
		emit(diagContractInvalid("", "Unsupported contract version %q; expected %q.", schema, jsonSchema2020_12))
	}

	_, rootTypePresent, typeOK, typeDiags := decodeTypeField(root, "type", "", rootTypeField)
	out = append(out, typeDiags...)
	if !typeOK && !rootTypePresent {
		emit(diagContractInvalid("", "Contract root must declare %q of %q.", "type", "object"))
	}

	title, td := decodeStringField(root, "title")
	out = append(out, td...)

	description, dd := decodeStringField(root, "description")
	out = append(out, dd...)

	required, rd := decodeRequiredField(root)
	out = append(out, rd...)

	properties, pd := decodePropertiesMap(root)
	out = append(out, pd...)

	// Normalize each variable.
	variables := make([]*Variable, 0, len(properties))
	for name := range properties {
		v, diags := normalizeVariable(name, properties[name])
		out = append(out, diags...)
		if v == nil {
			continue
		}
		variables = append(variables, v)
	}

	// Apply requiredness from the root required array.
	seenRequired := map[string]bool{}
	for _, name := range required {
		seenRequired[name] = true
	}
	for _, v := range variables {
		if seenRequired[v.Name] {
			v.Required = true
		}
	}

	// required/default conflict: requiredness is only known now that the root
	// required array has been applied.
	for _, v := range variables {
		if v.Required && v.HasDefault() {
			emit(diagContractInvalid(v.Name, "Variable %q is required and must not declare a default.", v.Name))
		}
	}

	// Contract-level required checks: every required name must be a declared
	// variable.
	declared := map[string]bool{}
	for _, v := range variables {
		declared[v.Name] = true
	}
	for _, name := range required {
		if !declared[name] {
			emit(diagContractInvalid(name, "Required variable %q is not declared in %q.", name, "properties"))
		}
	}

	if hasErrors(out) {
		return nil, sortDiagnostics(out)
	}

	sort.Slice(variables, func(i, j int) bool { return variables[i].Name < variables[j].Name })
	byName := make(map[string]*Variable, len(variables))
	for _, v := range variables {
		byName[v.Name] = v
	}

	return &Contract{
		Schema:      schema,
		Title:       title,
		Description: description,
		Variables:   variables,
		byName:      byName,
	}, sortDiagnostics(out)
}

func normalizeVariable(name string, raw json.RawMessage) (*Variable, []Diagnostic) {
	var out []Diagnostic
	emit := func(d Diagnostic) { out = append(out, d) }

	fields, err := decodeObjectFields(raw)
	if err != nil {
		var dup *duplicateKeyError
		if errors.As(err, &dup) {
			emit(diagContractInvalid(name, "Duplicate key %q in variable %q.", dup.key, name))
		} else {
			emit(diagContractInvalid(name, "Variable %q is not a valid JSON object: %v.", name, err))
		}
		return nil, out
	}
	for key := range fields {
		if !variableKeywords[key] {
			emit(diagUnsupportedKeyword(name, "Unsupported keyword %q in variable %q.", key, name))
		}
	}

	v := &Variable{Name: name}

	// type
	typ, typePresent, typeOK, typeDiags := decodeTypeField(fields, "type", name, variableTypeField)
	out = append(out, typeDiags...)
	if typeOK {
		v.Type = typ
	} else if !typePresent {
		emit(diagContractInvalid(name, "Variable %q must declare a valid %q (one of string, integer, number, boolean).", name, "type"))
	}

	// title / description
	v.Title, _ = decodeStringField(fields, "title")
	v.Description, _ = decodeStringField(fields, "description")

	// secret marker
	secret, secretDiags := decodeSecretField(fields)
	out = append(out, secretDiags...)
	v.Secret = secret

	// default
	defaultVal, defaultDiags := decodeDefaultField(fields, v.Type)
	out = append(out, defaultDiags...)
	v.Default = defaultVal

	// enum / const
	enumVals, enumDiags := decodeEnumField(fields, v.Type)
	out = append(out, enumDiags...)
	v.Enum = enumVals
	constVal, constDiags := decodeConstField(fields, v.Type)
	out = append(out, constDiags...)
	v.Const = constVal

	// pattern
	patternRaw, patternDiags := decodePatternField(fields)
	out = append(out, patternDiags...)
	if patternRaw != "" {
		re, err := regexp.Compile(patternRaw)
		if err != nil {
			emit(diagContractInvalid(name, "Invalid %q regex %q in variable %q: %v.", "pattern", patternRaw, name, err))
		} else {
			v.Pattern = re
			v.PatternRaw = patternRaw
		}
	}

	// string constraints
	minLength, minLengthDiags := decodeIntField(fields, "minLength", name)
	out = append(out, minLengthDiags...)
	v.MinLength = minLength
	maxLength, maxLengthDiags := decodeIntField(fields, "maxLength", name)
	out = append(out, maxLengthDiags...)
	v.MaxLength = maxLength

	// numeric constraints
	minimum, minimumDiags := decodeBoundField(fields, "minimum", name)
	out = append(out, minimumDiags...)
	v.Minimum = minimum
	maximum, maximumDiags := decodeBoundField(fields, "maximum", name)
	out = append(out, maximumDiags...)
	v.Maximum = maximum
	exclMin, exclMinDiags := decodeBoundField(fields, "exclusiveMinimum", name)
	out = append(out, exclMinDiags...)
	v.ExclusiveMinimum = exclMin
	exclMax, exclMaxDiags := decodeBoundField(fields, "exclusiveMaximum", name)
	out = append(out, exclMaxDiags...)
	v.ExclusiveMaximum = exclMax
	multipleOf, multipleOfDiags := decodePositiveNumberField(fields, "multipleOf", name)
	out = append(out, multipleOfDiags...)
	v.MultipleOf = multipleOf

	// format
	format, formatDiags := decodeFormatField(fields, name)
	out = append(out, formatDiags...)
	v.Format = format

	consistencyDiags := checkConstraintConsistency(v)
	out = append(out, consistencyDiags...)

	// type-appropriateness of constraints
	if v.Type != "" {
		out = append(out, checkConstraintTypes(v)...)
	}

	// secret/default conflict: a secret variable must not declare a default.
	if v.Secret && v.HasDefault() {
		emit(diagContractInvalid(name, "Secret variable %q must not declare a default.", name))
	}

	// secret usability: secret with enum/const forbidden? The spec forbids
	// secrets having defaults. Enum/const on secrets is not forbidden.
	if hasErrors(out) {
		return nil, out
	}

	// default value must satisfy declared constraints
	if v.HasDefault() {
		for _, problem := range v.describeConstraintViolations(v.Default) {
			emit(diagContractInvalid(name, "Default for variable %q violates its constraints: %s", name, problem))
		}
	}

	// const value must satisfy declared constraints
	if v.HasConst() {
		for _, problem := range v.describeConstraintViolations(v.Const) {
			emit(diagContractInvalid(name, "Const for variable %q violates its constraints: %s", name, problem))
		}
	}

	if hasErrors(out) {
		return nil, out
	}
	return v, out
}

// checkConstraintConsistency validates statically contradictory constraints.
func checkConstraintConsistency(v *Variable) []Diagnostic {
	var out []Diagnostic
	emit := func(d Diagnostic) { out = append(out, d) }

	if v.MinLength != nil && v.MaxLength != nil && *v.MinLength > *v.MaxLength {
		emit(diagContractInvalid(v.Name, "Variable %q declares minLength greater than maxLength.", v.Name))
	}
	if v.Minimum != nil && v.Maximum != nil && *v.Minimum > *v.Maximum {
		emit(diagContractInvalid(v.Name, "Variable %q declares minimum greater than maximum.", v.Name))
	}
	if v.ExclusiveMinimum != nil && v.Maximum != nil && *v.ExclusiveMinimum >= *v.Maximum {
		emit(diagContractInvalid(v.Name, "Variable %q declares exclusiveMinimum not below maximum; no value can satisfy it.", v.Name))
	}
	if v.Minimum != nil && v.ExclusiveMaximum != nil && *v.Minimum >= *v.ExclusiveMaximum {
		emit(diagContractInvalid(v.Name, "Variable %q declares minimum not below exclusiveMaximum; no value can satisfy it.", v.Name))
	}
	if v.ExclusiveMinimum != nil && v.ExclusiveMaximum != nil && *v.ExclusiveMinimum >= *v.ExclusiveMaximum {
		emit(diagContractInvalid(v.Name, "Variable %q declares exclusiveMinimum not below exclusiveMaximum; no value can satisfy it.", v.Name))
	}

	// const must be consistent with enum when both declared
	if v.HasConst() && len(v.Enum) > 0 {
		found := false
		for _, e := range v.Enum {
			if valuesEqual(e, v.Const) {
				found = true
				break
			}
		}
		if !found {
			emit(diagContractInvalid(v.Name, "Variable %q defines a const that is not a member of its enum.", v.Name))
		}
	}
	return out
}

// checkConstraintTypes verifies that a constraint applies to the declared type.
func checkConstraintTypes(v *Variable) []Diagnostic {
	var out []Diagnostic
	emit := func(d Diagnostic) { out = append(out, d) }

	hasNumeric := v.Minimum != nil || v.Maximum != nil || v.ExclusiveMinimum != nil ||
		v.ExclusiveMaximum != nil || v.MultipleOf != nil
	hasLength := v.MinLength != nil || v.MaxLength != nil

	switch v.Type {
	case TypeString:
		if hasNumeric {
			emit(diagContractInvalid(v.Name, "Numeric constraints are not allowed on %s variable %q.", TypeString, v.Name))
		}
	case TypeBoolean:
		if hasNumeric {
			emit(diagContractInvalid(v.Name, "Numeric constraints are not allowed on %s variable %q.", TypeBoolean, v.Name))
		}
		if hasLength {
			emit(diagContractInvalid(v.Name, "Length constraints are not allowed on %s variable %q.", TypeBoolean, v.Name))
		}
		if v.Pattern != nil {
			emit(diagContractInvalid(v.Name, "The %q constraint requires a %s variable; %q is %s.", "pattern", TypeString, v.Name, v.Type))
		}
		if v.Format != "" {
			emit(diagContractInvalid(v.Name, "The %q constraint requires a %s variable; %q is %s.", "format", TypeString, v.Name, v.Type))
		}
	default:
		if hasLength {
			emit(diagContractInvalid(v.Name, "Length constraints are not allowed on %s variable %q.", v.Type, v.Name))
		}
		if v.Pattern != nil {
			emit(diagContractInvalid(v.Name, "The %q constraint requires a %s variable; %q is %s.", "pattern", TypeString, v.Name, v.Type))
		}
		if v.Format != "" {
			emit(diagContractInvalid(v.Name, "The %q constraint requires a %s variable; %q is %s.", "format", TypeString, v.Name, v.Type))
		}
	}
	return out
}

// describeConstraintViolations returns human descriptions of every declared
// constraint the value violates. Used for the contract-level default check.
func (v *Variable) describeConstraintViolations(val Value) []string {
	var problems []string
	if len(v.Enum) > 0 && !v.EnumContains(val) {
		problems = append(problems, "value is not a member of the declared enum")
	}
	if v.HasConst() && !v.ConstMatches(val) {
		problems = append(problems, "value does not match the declared const")
	}
	if v.Type == TypeString {
		if s, ok := val.(string); ok {
			problems = append(problems, v.checkStringValue(s)...)
		}
	}
	if v.Type == TypeInteger || v.Type == TypeNumber {
		if !v.InNumericRange(val) {
			problems = append(problems, "value is outside the declared numeric range")
		}
	}
	return problems
}

// checkStringValue reports which string constraints the value violates.
func (v *Variable) checkStringValue(s string) []string {
	var problems []string
	if v.MinLength != nil && intLength(s) < *v.MinLength {
		problems = append(problems, fmt.Sprintf("value is shorter than minLength %d", *v.MinLength))
	}
	if v.MaxLength != nil && intLength(s) > *v.MaxLength {
		problems = append(problems, fmt.Sprintf("value is longer than maxLength %d", *v.MaxLength))
	}
	if v.Pattern != nil && !v.Pattern.MatchString(s) {
		problems = append(problems, "value does not match pattern")
	}
	return problems
}

func intLength(s string) int { return len([]rune(s)) }

// EnumContains reports whether val is an exact member of the declared enum.
func (v *Variable) EnumContains(val Value) bool {
	for _, e := range v.Enum {
		if valuesEqual(e, val) {
			return true
		}
	}
	return false
}

// ConstMatches reports whether val equals the declared const.
func (v *Variable) ConstMatches(val Value) bool {
	return v.HasConst() && valuesEqual(v.Const, val)
}

// InNumericRange reports whether val satisfies the declared numeric bounds.
func (v *Variable) InNumericRange(val Value) bool {
	if v.Minimum != nil && !numGE(val, *v.Minimum, false) {
		return false
	}
	if v.Maximum != nil && !numLE(val, *v.Maximum, false) {
		return false
	}
	if v.ExclusiveMinimum != nil && !numGE(val, *v.ExclusiveMinimum, true) {
		return false
	}
	if v.ExclusiveMaximum != nil && !numLE(val, *v.ExclusiveMaximum, true) {
		return false
	}
	if v.MultipleOf != nil && !multipleOf(val, *v.MultipleOf) {
		return false
	}
	return true
}

// InLengthRange reports whether the string length satisfies minLength/maxLength.
func (v *Variable) InLengthRange(s string) bool {
	n := intLength(s)
	if v.MinLength != nil && n < *v.MinLength {
		return false
	}
	if v.MaxLength != nil && n > *v.MaxLength {
		return false
	}
	return true
}

// numGE reports val >= bound, or val > bound when exclusive.
func numGE(val Value, bound float64, exclusive bool) bool {
	switch n := val.(type) {
	case int64:
		if b, ok := floatAsInt64(bound); ok {
			if exclusive {
				return n > b
			}
			return n >= b
		}
		if exclusive {
			return float64(n) > bound
		}
		return float64(n) >= bound
	case float64:
		if exclusive {
			return n > bound
		}
		return n >= bound
	default:
		return false
	}
}

// numLE reports val <= bound, or val < bound when exclusive.
func numLE(val Value, bound float64, exclusive bool) bool {
	switch n := val.(type) {
	case int64:
		if b, ok := floatAsInt64(bound); ok {
			if exclusive {
				return n < b
			}
			return n <= b
		}
		if exclusive {
			return float64(n) < bound
		}
		return float64(n) <= bound
	case float64:
		if exclusive {
			return n < bound
		}
		return n <= bound
	default:
		return false
	}
}

// floatAsInt64 converts a bound to int64 when it is integral and representable.
func floatAsInt64(f float64) (int64, bool) {
	if math.Trunc(f) != f {
		return 0, false
	}
	if f < -9.2233720368547758e18 || f >= 9.2233720368547758e18 {
		return 0, false
	}
	return int64(f), true
}

// multipleOf reports whether val is an exact multiple of m (> 0).
func multipleOf(val Value, m float64) bool {
	switch n := val.(type) {
	case int64:
		if mi, ok := floatAsInt64(m); ok && mi != 0 {
			return n%mi == 0
		}
		return modFloat(float64(n), m)
	case float64:
		return modFloat(n, m)
	default:
		return false
	}
}

// modFloat reports whether dividing n by m yields a mathematical integer using
// IEEE-754 double arithmetic, matching JSON Schema's multipleOf semantics.
func modFloat(n, m float64) bool {
	if m == 0 {
		return false
	}
	q := n / m
	return q == math.Trunc(q)
}

// valuesEqual compares two interpreted scalar values by Go identity of type.
func valuesEqual(a, b Value) bool {
	at, aok := a.(bool)
	bt, bok := b.(bool)
	if aok && bok {
		return at == bt
	}
	ai, aok := a.(int64)
	bi, bok := b.(int64)
	if aok && bok {
		return ai == bi
	}
	af, aok := a.(float64)
	bf, bok := b.(float64)
	if aok && bok {
		return af == bf
	}
	as, aok := a.(string)
	bs, bok := b.(string)
	if aok && bok {
		return as == bs
	}
	return false
}

// hasErrors reports whether any diagnostic has error severity.
func hasErrors(ds []Diagnostic) bool {
	for _, d := range ds {
		if d.Severity == diagnostic.SeverityError {
			return true
		}
	}
	return false
}

// sortDiagnostics returns diagnostics sorted deterministically.
func sortDiagnostics(ds []Diagnostic) []Diagnostic {
	sorted := make([]Diagnostic, len(ds))
	copy(sorted, ds)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Variable != sorted[j].Variable {
			return sorted[i].Variable < sorted[j].Variable
		}
		if sorted[i].Code != sorted[j].Code {
			return sorted[i].Code < sorted[j].Code
		}
		return sorted[i].Message < sorted[j].Message
	})
	return sorted
}
