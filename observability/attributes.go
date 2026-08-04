package observability

import (
	"go.opentelemetry.io/otel/attribute"
)

func toKeyValues(
	attributes []Attribute,
) []attribute.KeyValue {
	keyValues := make(
		[]attribute.KeyValue,
		0,
		len(attributes),
	)

	for _, attribute := range attributes {
		keyValues = append(
			keyValues,
			attribute.keyValue,
		)
	}

	return keyValues
}
