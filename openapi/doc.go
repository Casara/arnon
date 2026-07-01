/*
Package openapi provides OpenAPI 3.1 schema generation for HTTP endpoints.

The package follows a code-first approach and automatically infers schemas
from Go request/response types using struct tags such as:

	json
	path
	query
	header
	validate

It supports optional overrides through Operation definitions while keeping
the generated specification as automatic as possible.

OpenAPI generation is framework-agnostic and can be integrated with any
router or HTTP abstraction.
*/
package openapi
