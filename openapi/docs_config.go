package openapi

// DocsConfig configures Stoplight Elements.
type DocsConfig struct {
	Title string

	OpenAPIURL string

	FaviconURL string

	LogoURL string

	UseLocalAssets bool
}

func defaultDocsConfig() DocsConfig {
	return DocsConfig{
		Title: "API Documentation",

		OpenAPIURL: "/openapi.json",

		UseLocalAssets: false,
	}
}
