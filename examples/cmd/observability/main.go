// Command observability demonstrates arnon's OpenTelemetry
// integration: automatic HTTP spans (routing.WithInstrumentation),
// custom metrics (observability.Counter/Histogram) and
// trace-correlated structured logs, all exported over OTLP/gRPC to a
// collector.
//
// Run a local collector first - it prints every trace and metric it
// receives to its own stdout, so telemetry is visible without a
// separate backend (see otel-collector-config.yaml):
//
//	docker compose -f examples/cmd/observability/docker-compose.yml up
//
// Then, in another terminal:
//
//	go run ./examples/cmd/observability
//
// A request to POST /api/users produces a span (visible in the
// collector's logs almost immediately) and updates the
// users_created_total counter and create_user_duration_seconds
// histogram (visible after the periodic export interval, or
// immediately on graceful shutdown - Ctrl+C - which flushes both).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Casara/arnon/examples/internal/customvalidators"
	"github.com/Casara/arnon/examples/internal/logging"
	"github.com/Casara/arnon/examples/internal/users"
	"github.com/Casara/arnon/httpx"
	"github.com/Casara/arnon/httpx/middleware"
	"github.com/Casara/arnon/httpx/routing"
	"github.com/Casara/arnon/observability"
	"github.com/Casara/arnon/observability/otel"
	"github.com/Casara/arnon/openapi"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second
)

// userService wraps users.CreateUser with the custom metrics this
// example demonstrates: a counter for users successfully created and
// a histogram of how long the handler takes.
type userService struct {
	created  *observability.Counter
	duration *observability.Histogram
}

func (service *userService) CreateUser(
	ctx context.Context,
	request users.CreateUserRequest,
) (users.CreateUserResponse, error) {
	start := time.Now()

	response, err := users.CreateUser(ctx, request)

	service.duration.Record(
		ctx,
		time.Since(start).Seconds(),
	)

	if err != nil {
		return users.CreateUserResponse{}, fmt.Errorf("create user: %w", err)
	}

	service.created.Add(ctx, 1)

	return response, nil
}

func main() {
	logger := logging.NewLogger()

	customvalidators.RegisterCustomValidators()

	otlpEndpoint := getEnv("OTLP_ENDPOINT", "localhost:4317")

	otlpInsecure, err := strconv.ParseBool(getEnv("OTLP_INSECURE", "true"))
	if err != nil {
		slog.Error("parse OTLP_INSECURE", "error", err)
		os.Exit(1)
	}

	shutdownOTel, err := otel.Initialize(otel.Config{
		ServiceName:        "arnon-observability-example",
		ServiceVersion:     "0.0.0",
		Environment:        "development",
		TracesEnabled:      true,
		MetricsEnabled:     true,
		TraceOTLPEndpoint:  otlpEndpoint,
		MetricOTLPEndpoint: otlpEndpoint,
		OTLPInsecure:       otlpInsecure,
	})
	if err != nil {
		slog.Error("initialize OpenTelemetry", "error", err)
		os.Exit(1)
	}

	created, err := observability.NewCounter(
		"users_created_total",
		"Number of users successfully created.",
	)
	if err != nil {
		slog.Error("create users_created_total counter", "error", err)
		os.Exit(1)
	}

	duration, err := observability.NewHistogram(
		"create_user_duration_seconds",
		"Duration of the create user handler, in seconds.",
	)
	if err != nil {
		slog.Error("create create_user_duration_seconds histogram", "error", err)
		os.Exit(1)
	}

	service := &userService{
		created:  created,
		duration: duration,
	}

	generator := openapi.NewGenerator(openapi.Info{
		Title:   "Example API",
		Version: "1.0.0",
	})

	router := routing.NewRouter(
		routing.WithOpenAPI(openapi.NewRegistry(generator)),
		routing.WithInstrumentation(otel.NewHandler),
	)

	// Only RealIP/RequestID go through BuildChain here: Logging can't
	// join them in the same global chain, see the comment below.
	router.Use(middleware.BuildChain(middleware.ChainConfig{
		RealIP:    true,
		RequestID: true,
	})...)

	api := router.Group("/api")

	// Logging is scoped to /api, not global, on purpose: instrumentHandler
	// wraps each route inside the mux dispatch (see router.register), so
	// a route's span only exists once the mux has matched it. A global
	// Logging middleware (router.Use) runs before that dispatch and would
	// never observe a trace_id/span_id. Registering it per-route/group,
	// after instrumentation has already run for that route, is what lets
	// newRequestLogger's observability.TraceID/SpanID calls find one.
	api.Use(
		middleware.Logging(logger),
	)

	api.POST("/users", httpx.Endpoint(
		service.CreateUser,
		httpx.EndpointConfig{
			SuccessStatus: http.StatusCreated,
			OpenAPI: &openapi.Operation{
				Summary:       "Create a user",
				SuccessStatus: http.StatusCreated,
			},
		},
	))

	document := generator.Generate()

	router.GET("/openapi.json", openapi.NewHandler(&document))
	router.GET("/docs", openapi.NewDocsHandler(nil))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	os.Exit(run(server, shutdownOTel))
}

// run starts server and blocks until it exits or the process receives
// an interrupt/termination signal, then shuts both the server and
// OpenTelemetry down gracefully - which is what flushes any spans and
// metrics still buffered in the SDK to the collector before exit.
//
// It returns the process exit code instead of calling os.Exit itself,
// so main can call os.Exit after run's own deferred cleanup
// (signal.NotifyContext's stop, the shutdown timeout's cancel) has run.
func run(
	server *http.Server,
	shutdownOTel func(context.Context) error,
) int {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- server.ListenAndServe()
	}()

	exitCode := 0

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped", "error", err)

			exitCode = 1
		}
	case <-ctx.Done():
		slog.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer cancel()

	err := server.Shutdown(shutdownCtx)
	if err != nil {
		slog.Error("shut down HTTP server", "error", err)
	}

	err = shutdownOTel(shutdownCtx)
	if err != nil {
		slog.Error("shut down OpenTelemetry", "error", err)
	}

	return exitCode
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
