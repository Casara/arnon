package openapi

import "fmt"

// GeneratorOption configures a Generator during construction. Mirrors
// routing.Option's shape (httpx/routing/options.go) for consistency
// across the module.
type GeneratorOption func(*Generator)

// WithServers sets the document's top-level Servers.
func WithServers(servers ...Server) GeneratorOption {
	return func(generator *Generator) {
		generator.document.Servers = servers
	}
}

// WithExternalDocs sets the document's top-level ExternalDocs.
func WithExternalDocs(docs *ExternalDocs) GeneratorOption {
	return func(generator *Generator) {
		generator.document.ExternalDocs = docs
	}
}

// WithSpecVersion sets the OpenAPI Specification version the generated
// document declares. Defaults to SpecVersion32.
//
// Use SpecVersion31 when a consumer's tooling does not accept 3.2 yet - a
// client generator, a validator, or the documentation UI. Nothing this package
// generates uses a 3.2-only construct, so the document is byte-identical apart
// from the version string; only the declaration changes.
//
// An unrecognized version is rejected at construction rather than producing a
// document that claims conformance to something this package cannot guarantee.
func WithSpecVersion(version SpecVersion) GeneratorOption {
	return func(generator *Generator) {
		switch version {
		case SpecVersion31, SpecVersion32:
			generator.document.OpenAPI = version

		default:
			panic(fmt.Sprintf(
				"openapi: unsupported spec version %q, want %q or %q",
				version, SpecVersion31, SpecVersion32,
			))
		}
	}
}
