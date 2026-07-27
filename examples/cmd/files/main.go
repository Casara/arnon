// Command files demonstrates non-JSON HTTP content: file downloads (a
// static PDF, a streamed CSV, a streamed XML) and a multipart file
// upload with a validated content-type allow-list.
//
// None of this goes through httpx.Endpoint, which is JSON-only by
// design (see CLAUDE.md, "httpx.Endpoint is JSON-only by design").
// Every route here is a plain http.Handler mounted directly on the
// router, exactly like Endpoint's own doc comment says any other
// representation should be - so there's no OpenAPI generation in
// this example either: OpenAPI registration only happens for
// httpx.Endpoint routes. See examples/internal/files for the
// handlers themselves.
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Casara/arnon/examples/internal/files"
	"github.com/Casara/arnon/examples/internal/logging"
	"github.com/Casara/arnon/httpx/middleware"
	"github.com/Casara/arnon/httpx/routing"
)

const (
	readHeaderTimeout = 5 * time.Second

	maxUploadBodyBytes = 5 << 20 // 5 MiB
)

func main() {
	logging.NewLogger()

	router := routing.NewRouter()

	router.GET("/downloads/report.pdf", http.HandlerFunc(files.DownloadPDF))
	router.GET("/downloads/users.csv", http.HandlerFunc(files.DownloadCSV))
	router.GET("/downloads/users.xml", http.HandlerFunc(files.DownloadXML))

	uploads := router.Group("/uploads")

	// MaxBodyBytes bounds a multipart body exactly like it would bound
	// a JSON one - it works on the raw request, before
	// ParseMultipartForm ever runs, so an oversized upload is rejected
	// without holding its bytes in memory first.
	uploads.Use(middleware.MaxBodyBytes(maxUploadBodyBytes))

	uploads.POST("/file", http.HandlerFunc(files.Upload))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	err := server.ListenAndServe()
	if err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
