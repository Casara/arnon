package openapi

import (
	"reflect"
)

// SchemaGenerator derives an OpenAPI Schema from a Go value's type via
// reflection, reading `validate`, `description`, `format`, `example`
// and `default` struct tags along the way.
type SchemaGenerator struct{}

// NewSchemaGenerator creates a SchemaGenerator.
func NewSchemaGenerator() *SchemaGenerator {
	return &SchemaGenerator{}
}

// GenerateSchema builds the OpenAPI Schema for value's type.
func (generator *SchemaGenerator) GenerateSchema(
	value any,
) *Schema {
	return generator.generateType(
		reflect.TypeOf(value),
	)
}

func (generator *SchemaGenerator) generateType(
	valueType reflect.Type,
) *Schema {
	if valueType == nil {
		return &Schema{}
	}

	for valueType.Kind() == reflect.Pointer {
		valueType = valueType.Elem()
	}

	switch valueType.Kind() {
	case reflect.String:
		return &Schema{
			Type: "string",
		}

	case reflect.Bool:
		return &Schema{
			Type: "boolean",
		}

	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:
		return &Schema{
			Type: "integer",
		}

	case reflect.Float32, reflect.Float64:
		return &Schema{
			Type: "number",
		}

	case reflect.Slice, reflect.Array:
		return &Schema{
			Type: "array",
			Items: generator.generateType(
				valueType.Elem(),
			),
		}

	case reflect.Map:
		return &Schema{
			Type: "object",
			AdditionalProperties: generator.generateType(
				valueType.Elem(),
			),
		}

	case reflect.Struct:
		return generator.
			generateStruct(
				valueType,
			)

	default:
		// Unsupported Go types are represented as strings in the generated schema.
		return &Schema{
			Type: "string",
		}
	}
}

func (generator *SchemaGenerator) generateStruct(
	valueType reflect.Type,
) *Schema {
	schema := &Schema{
		Type: "object",
		Properties: make(
			map[string]*Schema,
		),
	}

	required := make(
		[]string,
		0,
	)

	for field := range valueType.Fields() {
		result := generator.parseField(
			field,
			&required,
		)

		if result == nil {
			continue
		}

		if result.Location != "body" {
			continue
		}

		schema.Properties[result.Name] = result.Schema
	}

	if len(required) > 0 {
		schema.Required = required
	}

	return schema
}
