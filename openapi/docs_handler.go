package openapi

import (
	"fmt"
	"net/http"
	"strings"
)

// DocsHandler serves Stoplight Elements.
type DocsHandler struct {
	config DocsConfig
}

// NewDocsHandler creates a docs handler.
func NewDocsHandler(config *DocsConfig) *DocsHandler {
	cfg := defaultDocsConfig()

	if config != nil {
		if config.Title != "" {
			cfg.Title = config.Title
		}

		if config.OpenAPIURL != "" {
			cfg.OpenAPIURL = config.OpenAPIURL
		}

		if config.FaviconURL != "" {
			cfg.FaviconURL = config.FaviconURL
		}

		if config.LogoURL != "" {
			cfg.LogoURL = config.LogoURL
		}
	}

	return &DocsHandler{
		config: cfg,
	}
}

// ServeHTTP serves API docs.
func (handler *DocsHandler) ServeHTTP(
	writer http.ResponseWriter,
	_ *http.Request,
) {
	writer.Header().Set(
		"Content-Type",
		"text/html; charset=utf-8",
	)

	_, _ = writer.Write(
		[]byte(
			buildHTML(
				handler.config,
			),
		),
	)
}

func buildHTML(
	config DocsConfig,
) string {
	scriptURL := "https://unpkg.com/@stoplight/elements/web-components.min.js"

	cssURL := "https://unpkg.com/@stoplight/elements/styles.min.css"

	faviconHTML := ""

	if config.FaviconURL != "" {
		faviconHTML = fmt.Sprintf(
			`<link rel="icon" href="%s" />`,
			config.FaviconURL,
		)
	}

	elementsAttributes := []string{
		fmt.Sprintf(
			`apiDescriptionUrl="%s"`,
			config.OpenAPIURL,
		),

		`layout="responsive"`,

		`router="hash"`,

		`tryItCredentialsPolicy="same-origin"`,
	}

	if config.LogoURL != "" {
		elementsAttributes = append(
			elementsAttributes,

			fmt.Sprintf(
				`logo="%s"`,
				config.LogoURL,
			),
		)
	}

	return fmt.Sprintf(
		docsHTMLTemplate,

		config.Title,

		faviconHTML,

		scriptURL,

		cssURL,

		strings.Join(
			elementsAttributes,
			"\n    ",
		),
	)
}

const docsHTMLTemplate = `
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="referrer" content="same-origin" />
  <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no" />
  <title>%s</title>
  %s
  <script src="%s"></script>
  <link rel="stylesheet" href="%s" />
</head>
<body style="height: 100vh; margin: 0;">
  <elements-api
    %s
  />
</body>
</html>
`
