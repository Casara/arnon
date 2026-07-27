// Command staticfiles demonstrates serving static assets
// (examples/cmd/staticfiles/assets: an HTML page and a stylesheet)
// through http.FileServer, mounted directly on the router like any
// other non-JSON content (see examples/cmd/files) - and, more
// importantly, why that route deliberately does NOT go through
// arnon's own ETag or Compress middleware, unlike /api/health below.
//
// http.FileServer (via http.ServeContent) already implements its own
// conditional GET (Last-Modified/If-Modified-Since) and HTTP Range
// support (resumable downloads, video/audio seeking, "Accept-Ranges:
// bytes"). Wrapping it in arnon's ETag/Compress breaks that, confirmed
// empirically by running both combinations and inspecting the actual
// response headers:
//
//   - ETag buffers the whole response and only computes/sets its Etag
//     header *after* the handler returns (httpx/middleware/etag.go).
//     By the time http.ServeContent runs (still inside the handler),
//     no Etag exists yet for it to compare an incoming If-Range
//     against, so If-Range silently fails to match and the server
//     falls back to a full 200 response instead of the expected 206 -
//     exactly the resume-a-download case If-Range exists for. Worse,
//     the Etag arnon does end up setting is a hash of whatever bytes
//     were written for *that specific request* - a 100-byte Range
//     response and the full file produce two different Etags for the
//     same resource, which breaks the "stable validator" assumption
//     conditional requests rely on.
//   - Compress gzips whatever bytes the handler wrote and deletes
//     Content-Length (correct for a full 200 body) but never touches
//     Content-Range, which http.ServeContent had already set to
//     describe byte positions in the *uncompressed* resource. The
//     result (confirmed empirically): a 206 response advertising
//     e.g. "Content-Range: bytes 0-99/10000" while the actual body on
//     the wire is a self-contained gzip stream of a different length -
//     internally consistent enough that a naive client can still
//     decompress it, but not what Content-Range is specified to mean,
//     and not something a real Range-aware client (a download
//     manager resuming by byte offset) can rely on.
//
// The fix isn't a framework change: it's route scoping, the same
// technique examples/cmd/middleware already uses for
// AllowContentType/MaxBodyBytes (global middleware isn't the only
// option - Group.Use scopes middleware to exactly the routes it makes
// sense for). /api/health below is a normal JSON endpoint that never
// serves Range-capable content, so ETag there is both safe and
// useful; /static/ is not, so it gets neither ETag nor Compress -
// only http.FileServer's own, already-correct conditional GET/Range
// handling.
package main

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Casara/arnon/examples/internal/logging"
	"github.com/Casara/arnon/httpx"
	"github.com/Casara/arnon/httpx/middleware"
	"github.com/Casara/arnon/httpx/routing"
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

	// No ETag, no Compress - see the package doc comment above for
	// why combining either with http.FileServer's own Range support
	// is actively harmful, not just redundant.
	router.GET(
		"/static/",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.FS(staticAssets)),
		),
	)

	api := router.Group("/api")

	// Safe here: /api/health never serves Range-capable content, so
	// ETag's buffer-then-hash approach has nothing to conflict with.
	api.Use(middleware.ETag())

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
