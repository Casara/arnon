package openapi

import "reflect"

// Registry stores OpenAPI schema definitions and guarantees that
// each schema is registered only once.
type Registry struct {
	generator *Generator
}

// NewRegistry creates an empty schema registry.
func NewRegistry(
	generator *Generator,
) *Registry {
	return &Registry{
		generator: generator,
	}
}

func (registry *Registry) Register(
	method string,
	path string,
	operation Operation,
	requestType reflect.Type,
	responseType reflect.Type,
) {
	registry.generator.RegisterTypes(
		method,
		path,
		operation,
		requestType,
		responseType,
	)
}
