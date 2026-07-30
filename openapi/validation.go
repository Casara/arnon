package openapi

import (
	"strconv"
	"strings"

	"github.com/casara/arnon/validation"
)

// applyValidationTags translates a `validate` struct tag into OpenAPI
// schema constraints, one switch arm per validator tag, mirroring the
// mapping in validation.mapFieldError so both stay easy to compare.
//
// A `dive` tag (go-playground/validator's marker for "validate each
// element of this slice/array/map instead of the field itself")
// retargets every constraint tag that follows it from schema to
// schema.Items - without this, `validate:"dive,min=2"` on a
// `[]string` field would set MinItems (array must have >= 2 elements)
// instead of MinLength on each element (each string must have >= 2
// characters), because applyMin/applyMax only look at the target
// schema's Type, which is "array" either way. Repeated `dive` tags
// (a slice of slices) descend one further Items level each time.
//
//nolint:cyclop,funlen // one switch arm per validator tag, see comment above
func applyValidationTags(
	schema *Schema,
	tagValue string,
	required *[]string,
	fieldName string,
) {
	if tagValue == "" {
		return
	}

	tags := strings.SplitSeq(
		tagValue,
		",",
	)

	target := schema
	dived := false

	for tag := range tags {
		name, value, hasValue := strings.Cut(tag, "=")

		if name == "dive" {
			dived = true

			if target.Items != nil {
				target = target.Items
			}

			continue
		}

		switch name {
		case "required":
			// A dived "required" means each element must be non-zero,
			// which has no OpenAPI object-level "required" equivalent
			// (that lists property names, not an array-content rule) -
			// only a pre-dive "required" marks the field itself required.
			if !dived {
				*required = append(
					*required,
					fieldName,
				)
			}

		case "min":
			applyMin(
				target,
				value,
			)

		case "max":
			applyMax(
				target,
				value,
			)

		case "gt":
			applyGT(
				target,
				value,
			)

		case "gte":
			applyGTE(
				target,
				value,
			)

		case "lt":
			applyLT(
				target,
				value,
			)

		case "lte":
			applyLTE(
				target,
				value,
			)

		case "len":
			applyLen(
				target,
				value,
			)

		case "oneof":
			if hasValue {
				applyOneOf(
					target,
					value,
				)
			}

		case "email":
			applyEmail(target)

		case "uuid":
			applyUUID(target)

		case "url":
			applyURL(target)

		default:
			applyCustomRule(target, name)
		}
	}
}

// applyCustomRule enriches schema using the OpenAPI effect declared by
// a framework-registered custom validation rule, if any. Tags that
// have no custom rule registered are left untouched, preserving
// whatever the schema generator inferred on its own.
func applyCustomRule(schema *Schema, tag string) {
	rule, ok := validation.LookupCustomRule(tag)
	if !ok || rule.Schema == nil {
		return
	}

	if rule.Schema.Format != "" {
		schema.Format = rule.Schema.Format
	}

	if rule.Schema.Pattern != "" {
		schema.Pattern = rule.Schema.Pattern
	}
}

func applyMin(
	schema *Schema,
	value string,
) {
	number, err := strconv.Atoi(
		value,
	)
	if err != nil {
		return
	}

	switch schema.Type {
	case "string":
		schema.MinLength = &number

	case "array":
		schema.MinItems = &number

	default:
		floatValue := float64(number)

		schema.Minimum = &floatValue
	}
}

func applyMax(
	schema *Schema,
	value string,
) {
	number, err := strconv.Atoi(
		value,
	)
	if err != nil {
		return
	}

	switch schema.Type {
	case "string":
		schema.MaxLength = &number

	case "array":
		schema.MaxItems = &number

	default:
		floatValue := float64(number)

		schema.Maximum = &floatValue
	}
}

func applyGT(
	schema *Schema,
	value string,
) {
	number, err := strconv.ParseFloat(
		value,
		64,
	)
	if err != nil {
		return
	}

	schema.ExclusiveMinimum = &number
}

func applyGTE(
	schema *Schema,
	value string,
) {
	number, err := strconv.ParseFloat(
		value,
		64,
	)
	if err != nil {
		return
	}

	schema.Minimum = &number
}

func applyLT(
	schema *Schema,
	value string,
) {
	number, err := strconv.ParseFloat(
		value,
		64,
	)
	if err != nil {
		return
	}

	schema.ExclusiveMaximum = &number
}

func applyLTE(
	schema *Schema,
	value string,
) {
	number, err := strconv.ParseFloat(
		value,
		64,
	)
	if err != nil {
		return
	}

	schema.Maximum = &number
}

func applyLen(
	schema *Schema,
	value string,
) {
	number, err := strconv.Atoi(
		value,
	)
	if err != nil {
		return
	}

	switch schema.Type {
	case "string":
		schema.MinLength = &number

		schema.MaxLength = &number

	case "array":
		schema.MinItems = &number

		schema.MaxItems = &number
	}
}

func applyOneOf(
	schema *Schema,
	value string,
) {
	if value == "" {
		return
	}

	options := strings.Fields(
		value,
	)

	enum := make(
		[]any,
		0,
		len(options),
	)

	for _, option := range options {
		enum = append(
			enum,
			option,
		)
	}

	schema.Enum = enum
}

func applyEmail(
	schema *Schema,
) {
	schema.Format = "email"
}

func applyUUID(
	schema *Schema,
) {
	schema.Format = "uuid"
}

func applyURL(
	schema *Schema,
) {
	schema.Format = "uri"
}
