// Command staticfiles demonstrates serving static assets
// (examples/cmd/staticfiles/assets: an HTML page and a stylesheet)
// through http.FileServer, mounted directly on the router like any
// other non-JSON content (see examples/cmd/files).
//
// http.FileServer (via http.ServeContent) implements its own
// conditional GET (Last-Modified/If-Modified-Since) and HTTP Range
// support (resumable downloads, video/audio seeking,
// "Accept-Ranges: bytes"). ETag and Compress are applied globally
// here, including to /static/: both step aside for any request
// carrying a Range header (see their doc comments in
// httpx/middleware/etag.go and compress.go for the mechanism and why
// it matters), so a plain GET to /static/ still gets a validator and
// gzip, while a Range GET gets neither and falls through untouched to
// http.ServeContent's own Range/conditional-GET handling.
//
// /api/health is the contrast case: a normal JSON endpoint that never
// serves Range-capable content, so the same global ETag applies to it
// exactly as intended, with no exceptions needed either way.
package main

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/casara/arnon/httpx"
	"github.com/casara/arnon/httpx/middleware"
	"github.com/casara/arnon/httpx/routing"

	"github.com/casara/arnon/examples/internal/logging"
)

//go:embed assets
var assetsFS embed.FS

const readHeaderTimeout = 5 * time.Second

// HealthResponse is the response body for Health.
type HealthResponse struct {
	Status string `json:"status"`
}

// Health is a minimal JSON endpoint with no request body, standing in
// for "a typical API route" - the point of contrast with /static/,
// not something interesting on its own.
func Health(
	_ context.Context,
	_ struct{},
) (HealthResponse, error) {
	return HealthResponse{Status: "ok"}, nil
}

func main() {
	logging.NewLogger()

	staticAssets, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		slog.Error("static assets", "error", err)
		os.Exit(1)
	}

	router := routing.NewRouter()

	// Global: applies to /static/ and /api/health alike. Neither
	// middleware needs scoping away from /static/ - see the package
	// doc comment above.
	router.Use(
		middleware.ETag(),
		middleware.Compress(),
	)

	router.GET(
		"/static/",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.FS(staticAssets)),
		),
	)

	api := router.Group("/api")

	api.GET("/health", httpx.Endpoint(
		Health,
		httpx.EndpointConfig{},
	))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	err = server.ListenAndServe()
	if err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
