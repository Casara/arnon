package openapi

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
