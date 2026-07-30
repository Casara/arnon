package validation

import "reflect"

// FieldContext is what a CustomRule sees when it runs: the value under
// validation, the tag's parameter, and the structs it sits inside.
//
// It is an interface rather than a struct so that the underlying validator's
// own field type satisfies it structurally, with no adapter and no conversion
// - and so that arnon's public API names none of that library's types. A rule
// written against this signature keeps compiling if the implementation behind
// validation.Default() is ever replaced.
type FieldContext interface {
	// Field is the value being validated. Use Field().String(),
	// Field().Int() and friends according to the kind you expect.
	Field() reflect.Value

	// Param is the text after "=" in the tag: "10" in `validate:"cpf=10"`,
	// empty for a bare `validate:"cpf"`.
	Param() string

	// Parent is the struct immediately containing the field, which is what a
	// cross-field rule needs to read a sibling.
	Parent() reflect.Value

	// Top is the outermost struct validation started from. It differs from
	// Parent once the field is nested.
	Top() reflect.Value
}

// RuleFunc reports whether a value satisfies a CustomRule. Returning false
// produces the rule's Code and Message as a problem.ValidationError.
//
// It must not panic and must not mutate the value: validation runs after
// sanitization precisely so that transforming and checking stay separate.
type RuleFunc func(field FieldContext) bool
