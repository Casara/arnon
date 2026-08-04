package validation

import (
	"github.com/casara/arnon/problem"
)

// SchemaEffect describes how a custom validation rule augments the
// OpenAPI schema generated for a field using its tag.
//
// Fields left as zero values are not applied, leaving the schema
// generator's own inference in place.
type SchemaEffect struct {
	// Format sets the OpenAPI "format" keyword, e.g. "cpf".
	Format string

	// Pattern sets the OpenAPI "pattern" keyword as a regular
	// expression.
	Pattern string
}

// CustomRule registers a custom validation tag with the framework.
//
// A single rule definition is applied consistently across the three
// surfaces that need to agree on what a tag means: runtime validation
// (Func), OpenAPI schema generation (Schema) and problem detail error
// mapping (Message and Code). Without this, a custom tag would
// validate correctly but produce no schema metadata and a generic
// error message.
type CustomRule struct {
	// Tag is the validator tag name, e.g. "cpf".
	Tag string

	// Func performs the runtime validation and is registered on every
	// validator instance created through New. See RuleFunc and FieldContext.
	Func RuleFunc

	// Schema optionally augments the generated OpenAPI schema for
	// fields using this tag.
	Schema *SchemaEffect

	// Code is the problem detail error code reported when validation
	// fails for this tag. Defaults to
	// problem.ValidationCodeValidationFailed when empty.
	Code problem.ValidationErrorCode

	// Message builds the human-readable detail message for a
	// validation failure, given the tag parameter (the part after
	// "=", e.g. "10" in "cpf=10"). Defaults to a generic message
	// when nil.
	Message func(param string) string
}
