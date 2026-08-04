package openapi

import (
	"reflect"
	"strconv"
	"strings"
)

type fieldResult struct {
	Name string

	Schema *Schema

	Location string

	Required bool
}

func (generator *SchemaGenerator) parseField(
	field reflect.StructField,
	required *[]string,
) *fieldResult {
	if !field.IsExported() {
		return nil
	}

	location, name := getFieldLocation(field)

	if name == "" {
		return nil
	}

	schema := generator.generateType(field.Type)

	if field.Type.Kind() == reflect.Pointer {
		schema.Nullable = true
	}

	fieldRequired := false

	localRequired := make([]string, 0)

	applyValidationTags(
		schema,
		field.Tag.Get(
			"validate",
		),
		&localRequired,
		name,
	)

	if len(localRequired) > 0 {
		fieldRequired = true

		*required = append(
			*required,
			name,
		)
	}

	schema.Description = field.Tag.Get("description")

	// An explicit `format` tag always wins. Otherwise, keep whatever
	// applyValidationTags already inferred from the `validate` tag
	// (including custom rules) before falling back to the additional
	// formats inferFormatFromValidator knows about.
	if explicitFormat := field.Tag.Get("format"); explicitFormat != "" {
		schema.Format = explicitFormat
	}

	if schema.Format == "" {
		schema.Format = inferFormatFromValidator(
			field.Tag.Get(
				"validate",
			),
		)
	}

	schema.Example = parseExample(field)
	schema.Default = parseDefault(field)
	schema.Deprecated = field.Tag.Get("deprecated") == "true"
	schema.ReadOnly = field.Tag.Get("readonly") == "true"
	schema.WriteOnly = field.Tag.Get("writeonly") == "true"

	return &fieldResult{
		Name:     name,
		Schema:   schema,
		Location: location,
		Required: fieldRequired,
	}
}

func getFieldLocation(
	field reflect.StructField,
) (
	string, // location
	string, // name
) {
	switch {
	case hasTag(field, "path"):
		return "path",
			field.Tag.Get(
				"path",
			)

	case hasTag(field, "query"):
		return "query",
			field.Tag.Get(
				"query",
			)

	case hasTag(field, "header"):
		return "header",
			field.Tag.Get(
				"header",
			)

	default:
		return "body",
			getJSONName(
				field,
			)
	}
}

func hasTag(
	field reflect.StructField,
	tag string,
) bool {
	value := strings.TrimSpace(
		field.Tag.Get(
			tag,
		),
	)

	return value != ""
}

func inferFormatFromValidator(validateTag string) string {
	if validateTag == "" {
		return ""
	}

	validations := strings.SplitSeq(validateTag, ",")

	for validation := range validations {
		name := validation

		if before, _, ok := strings.Cut(
			validation,
			"=",
		); ok {
			name = before
		}

		switch name {
		case "email":
			return "email"

		case "uuid", "uuid4":
			return "uuid"

		case "url", "uri":
			return "uri"

		case "hostname":
			return "hostname"

		case "ipv4":
			return "ipv4"

		case "ipv6":
			return "ipv6"

		case "ip":
			return "ip"

		case "datetime":
			return "date-time"

		case "date":
			return "date"
		}
	}

	return ""
}

func parseExample(field reflect.StructField) any {
	return parseTypedValue(
		field.Tag.Get("example"),
		field.Type,
	)
}

func parseDefault(field reflect.StructField) any {
	return parseTypedValue(
		field.Tag.Get("default"),
		field.Type,
	)
}

func parseTypedValue(value string, fieldType reflect.Type) any {
	if value == "" {
		return nil
	}

	for fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
	}

	//nolint:exhaustive // Only supported kinds are mapped.
	switch fieldType.Kind() {
	case reflect.String:
		return value

	case reflect.Bool:
		value, err := strconv.ParseBool(value)
		if err == nil {
			return value
		}

	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:
		value, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return value
		}

	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:
		value, err := strconv.ParseUint(value, 10, 64)
		if err == nil {
			return value
		}

	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return value
		}
	}

	return value
}
