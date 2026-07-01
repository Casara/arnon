package observability

import (
	"go.opentelemetry.io/otel/attribute"
)

// Attribute should only contain low-cardinality values.
//
// Good:
//
//	tenant=tier1
//	operation=create_category
//	status=success
//
// Bad:
//
//	user_id=123
//	request_id=abc
//	trace_id=xyz
//
// High-cardinality attributes can severely impact metrics backends.
type Attribute struct {
	keyValue attribute.KeyValue
}

// String creates a string attribute.
func String(
	key string,
	value string,
) Attribute {
	return Attribute{
		keyValue: attribute.String(
			key,
			value,
		),
	}
}

// Int creates an integer attribute.
func Int(
	key string,
	value int,
) Attribute {
	return Attribute{
		keyValue: attribute.Int(
			key,
			value,
		),
	}
}

// Bool creates a boolean attribute.
func Bool(
	key string,
	value bool,
) Attribute {
	return Attribute{
		keyValue: attribute.Bool(
			key,
			value,
		),
	}
}
