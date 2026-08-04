package openapi

// defaultElementsVersion pins the Stoplight Elements build the documentation
// UI loads.
//
// Pinned rather than tracking latest: an unversioned unpkg URL means the
// shipped UI can change - or break - with no commit here, and the bytes a
// browser executes would not be the ones this release was checked against.
const defaultElementsVersion = "9.0.24"

// DocsConfig configures the Stoplight Elements documentation UI. The zero
// value works: NewDocsHandler fills in every field left empty.
type DocsConfig struct {
	// Title is the HTML page title. Defaults to "API Documentation".
	Title string

	// OpenAPIURL is the path the UI fetches the document from. It has to be
	// reachable from the browser, so it is a URL on your own server, not a
	// filesystem path. Defaults to "/openapi.json".
	OpenAPIURL string

	// FaviconURL, when set, adds a favicon link to the page.
	FaviconURL string

	// LogoURL, when set, shows a logo in the UI's sidebar.
	LogoURL string

	// ElementsVersion selects the Stoplight Elements release loaded from
	// unpkg. Defaults to a version this release was checked against.
	//
	// Worth knowing before changing it: Stoplight Elements documents support
	// for OpenAPI up to 3.1, while Generator emits 3.2.0 by default. Nothing
	// generated here uses a 3.2-only construct, so it renders - but if a newer
	// Elements build ever rejects the version string outright, the fix is
	// openapi.WithSpecVersion(openapi.SpecVersion31) on the generator, not a
	// different UI. Re-check the rendered page in a browser after changing
	// this.
	ElementsVersion string
}

func (config DocsConfig) withDefaults() DocsConfig {
	if config.Title == "" {
		config.Title = "API Documentation"
	}

	if config.OpenAPIURL == "" {
		config.OpenAPIURL = "/openapi.json"
	}

	if config.ElementsVersion == "" {
		config.ElementsVersion = defaultElementsVersion
	}

	return config
}
