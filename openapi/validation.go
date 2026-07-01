package openapi

import (
	"strconv"
	"strings"
)

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

	for tag := range tags {
		name, value, hasValue := strings.Cut(tag, "=")

		switch name {
		case "required":
			*required = append(
				*required,
				fieldName,
			)

		case "min":
			applyMin(
				schema,
				value,
			)

		case "max":
			applyMax(
				schema,
				value,
			)

		case "gt":
			applyGT(
				schema,
				value,
			)

		case "gte":
			applyGTE(
				schema,
				value,
			)

		case "lt":
			applyLT(
				schema,
				value,
			)

		case "lte":
			applyLTE(
				schema,
				value,
			)

		case "len":
			applyLen(
				schema,
				value,
			)

		case "oneof":
			if hasValue {
				applyOneOf(
					schema,
					value,
				)
			}

		case "email":
			applyEmail(schema)

		case "uuid":
			applyUUID(schema)

		case "url":
			applyURL(schema)
		}
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
