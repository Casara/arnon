package openapi

import (
	"net/http"
	"reflect"
	"strconv"
)

// Generator accumulates registered endpoints into an OpenAPI Document.
//
// Each call to Register or RegisterTypes adds one path/operation and
// generates its request/response schemas; Generate returns the final
// Document once every route has been registered.
type Generator struct {
	document Document

	schemaGenerator *SchemaGenerator

	tags map[string]Tag
}

// NewGenerator creates a Generator that will produce a Document
// carrying the given API metadata. opts can set additional
// document-level fields (e.g. WithServers, WithExternalDocs) that
// have no dedicated constructor parameter of their own.
func NewGenerator(info Info, opts ...GeneratorOption) *Generator {
	generator := &Generator{
		document: Document{
			OpenAPI: OpenAPIVersion3_2,

			Info: info,

			Paths: make(
				map[string]PathItem,
			),

			Components: Components{
				Schemas: make(
					map[string]Schema,
				),
			},
		},

		schemaGenerator: NewSchemaGenerator(),

		tags: make(
			map[string]Tag,
		),
	}

	for _, opt := range opts {
		opt(generator)
	}

	return generator
}

// Register registers an endpoint.
func (generator *Generator) Register(
	method string,
	path string,
	operation Operation,
	request any,
	response any,
) {
	pathItem := generator.document.Paths[path]

	generator.buildOperation(
		method,
		&operation,
		request,
		response,
	)

	switch method {
	case http.MethodGet:
		pathItem.Get = &operation

	case http.MethodPost:
		pathItem.Post = &operation

	case http.MethodPut:
		pathItem.Put = &operation

	case http.MethodPatch:
		pathItem.Patch = &operation

	case http.MethodDelete:
		pathItem.Delete = &operation

	case http.MethodHead:
		pathItem.Head = &operation

	case http.MethodOptions:
		pathItem.Options = &operation

	case http.MethodTrace:
		pathItem.Trace = &operation

	// "QUERY" mirrors routing.MethodQuery (httpx/routing) - openapi
	// cannot import routing (see .go-arch-lint.yml), so the method name
	// is duplicated here as a literal instead.
	case "QUERY":
		pathItem.Query = &operation
	}

	generator.document.Paths[path] = pathItem
}

// RegisterTypes registers an endpoint from its request/response
// reflect.Type instead of live values, so callers that only have type
// information at route-registration time (e.g. the router) do not
// need to construct zero values themselves.
func (generator *Generator) RegisterTypes(
	method string,
	path string,
	operation Operation,
	requestType reflect.Type,
	responseType reflect.Type,
) {
	request := reflect.New(
		requestType,
	).Elem().
		Interface()

	response := reflect.New(
		responseType,
	).Elem().
		Interface()

	generator.Register(
		method,
		path,
		operation,
		request,
		response,
	)
}

// Generate generates the document.
func (generator *Generator) Generate() Document {
	document := generator.document

	document.Tags = make(
		[]Tag,
		0,
		len(generator.tags),
	)

	for _, tag := range generator.tags {
		document.Tags = append(
			document.Tags,
			tag,
		)
	}

	return document
}

func (generator *Generator) buildOperation(
	method string,
	operation *Operation,
	request any,
	response any,
) {
	generator.registerTags(operation.Tags)

	operation.TagNames = make(
		[]string,
		0,
		len(operation.Tags),
	)

	for _, tag := range operation.Tags {
		operation.TagNames = append(
			operation.TagNames,
			tag.Name,
		)
	}

	generator.addParameters(operation, request)

	if shouldGenerateBody(method, request) {
		requestRef := generator.registerSchema(request)

		operation.RequestBody = &RequestBody{
			Required: true,

			Content: map[string]MediaType{
				"application/json": {
					Schema: &Schema{
						Ref: requestRef,
					},
				},
			},
		}
	}

	responseRef := generator.registerSchema(response)

	if operation.Responses == nil {
		operation.Responses = make(Responses)
	}

	if len(operation.Responses) == 0 {
		statusCode := operation.SuccessStatus

		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		operation.Responses[strconv.Itoa(
			statusCode,
		)] = Response{
			Description: http.StatusText(
				statusCode,
			),

			Content: map[string]MediaType{
				"application/json": {
					Schema: &Schema{
						Ref: responseRef,
					},
				},
			},
		}

		generator.registerProblemSchema()

		problemRef := schemaRef("Problem")

		addDefaultProblemResponse(
			operation,
			"400",
			"Bad Request",
			problemRef,
			validationProblemExample(),
		)

		addDefaultProblemResponse(
			operation,
			"500",
			"Internal Server Error",
			problemRef,
			internalServerErrorExample(),
		)
	}
}

func (generator *Generator) addParameters(operation *Operation, request any) {
	requestType := reflect.TypeOf(request)

	if requestType == nil {
		return
	}

	for requestType.Kind() == reflect.Pointer {
		requestType = requestType.Elem()
	}

	if requestType.Kind() != reflect.Struct {
		return
	}

	required := make([]string, 0)

	for field := range requestType.Fields() {
		result := generator.schemaGenerator.parseField(field, &required)

		if result == nil {
			continue
		}

		if result.Location == "body" {
			continue
		}

		parameter := Parameter{
			Name: result.Name,

			In: result.Location,

			Required: result.Required || result.Location == "path",

			Schema: result.Schema,
		}

		operation.Parameters = append(operation.Parameters, parameter)
	}
}

func shouldGenerateBody(method string, request any) bool {
	if method == http.MethodGet {
		return false
	}

	requestType := reflect.TypeOf(request)

	if requestType == nil {
		return false
	}

	for requestType.Kind() == reflect.Pointer {
		requestType = requestType.Elem()
	}

	if requestType.Kind() != reflect.Struct {
		return false
	}

	for field := range requestType.Fields() {
		location, _ := getFieldLocation(
			field,
		)

		if location == "body" {
			return true
		}
	}

	return false
}

func (generator *Generator) registerSchema(value any) string {
	valueType := reflect.TypeOf(value)

	if valueType == nil {
		return ""
	}

	for valueType.Kind() == reflect.Pointer {
		valueType = valueType.Elem()
	}

	name := valueType.Name()

	if _, exists := generator.document.
		Components.
		Schemas[name]; exists {
		return schemaRef(
			name,
		)
	}

	schema := generator.
		schemaGenerator.
		GenerateSchema(
			value,
		)

	generator.document.
		Components.
		Schemas[name] = *schema

	return schemaRef(name)
}

func schemaRef(name string) string {
	return "#/components/schemas/" + name
}

func addDefaultProblemResponse(
	operation *Operation,
	status string,
	description string,
	problemRef string,
	example any,
) {
	if _, exists := operation.Responses[status]; exists {
		return
	}

	operation.Responses[status] = Response{
		Description: description,

		Content: map[string]MediaType{
			"application/problem+json": {
				Schema: &Schema{
					Ref: problemRef,
				},

				Example: example,
			},
		},
	}
}

func (generator *Generator) registerTags(
	tags []Tag,
) {
	for _, tag := range tags {
		if tag.Name == "" {
			continue
		}

		existing, exists := generator.tags[tag.Name]

		if !exists {
			generator.tags[tag.Name] = tag

			continue
		}

		if existing.Summary == "" {
			existing.Summary = tag.Summary
		}

		if existing.Description == "" {
			existing.Description = tag.Description
		}

		if existing.Parent == "" {
			existing.Parent = tag.Parent
		}

		if existing.Kind == "" {
			existing.Kind = tag.Kind
		}

		if existing.ExternalDocs == nil {
			existing.ExternalDocs = tag.ExternalDocs
		}

		generator.tags[tag.Name] = existing
	}
}

func validationProblemExample() map[string]any {
	return map[string]any{
		"title":  "Request validation failed",
		"status": http.StatusBadRequest,
		"detail": "Request validation failed",
		"errors": []any{
			map[string]any{
				"detail": "field is required",
				"code":   "required",
				"source": map[string]any{
					"in":    "body",
					"field": "/name",
				},
			},
		},
	}
}

func internalServerErrorExample() map[string]any {
	return map[string]any{
		"title":  "Internal Server Error",
		"status": http.StatusInternalServerError,
		"detail": "Unexpected error",
	}
}
