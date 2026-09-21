// Package contract implements the EnvDoctor Contract Profile: a restricted
// JSON Schema Draft 2020-12 subset for describing environment-variable
// requirements.
//
// A valid contract parses into the normalized Contract representation below.
// Arbitrary JSON Schema structures are never passed through the application:
// unsupported or invalid constructs produce contract diagnostics instead.
package contract

import (
	"regexp"
)

// Type is an EnvDoctor environment-variable primitive type.
type Type string

// TypeObject is the contract root type. object is a valid root type but is not
// a supported environment-variable type. array and null are not MVP types.
const TypeObject Type = "object"

// Supported variable types.
const (
	TypeString  Type = "string"
	TypeInteger Type = "integer"
	TypeNumber  Type = "number"
	TypeBoolean Type = "boolean"
)

// Value is an interpreted scalar variable value: one of string, int64,
// float64, or bool, matching the variable Type.
type Value = any

// Supported format identifiers.
const (
	FormatEmail = "email"
	FormatURI   = "uri"
)

// Variable describes one declared configuration variable.
type Variable struct {
	Name        string
	Type        Type
	Required    bool
	Secret      bool
	Description string
	Title       string

	Default Value   // nil when no default is declared
	Enum    []Value // nil when no enum is declared; values already type-interpreted
	Const   Value   // nil when no const is declared

	Pattern    *regexp.Regexp // nil when no pattern is declared
	PatternRaw string

	MinLength        *int
	MaxLength        *int
	Minimum          *float64
	Maximum          *float64
	ExclusiveMinimum *float64
	ExclusiveMaximum *float64
	MultipleOf       *float64

	Format string
}

// HasDefault reports whether the variable declares a default.
func (v *Variable) HasDefault() bool { return v.Default != nil }

// HasConst reports whether the variable declares a const.
func (v *Variable) HasConst() bool { return v.Const != nil }

// Contract is the normalized EnvDoctor configuration contract.
type Contract struct {
	Schema      string // declared $schema value
	Title       string
	Description string
	Variables   []*Variable // sorted by variable name
	byName      map[string]*Variable
}

// Variable returns the declared variable with the given name, or nil.
func (c *Contract) Variable(name string) *Variable {
	if c == nil {
		return nil
	}
	return c.byName[name]
}

// VariableNames returns declared variable names in deterministic order.
func (c *Contract) VariableNames() []string {
	if c == nil {
		return nil
	}
	names := make([]string, 0, len(c.Variables))
	for _, v := range c.Variables {
		names = append(names, v.Name)
	}
	return names
}
