package observability_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/Casara/arnon/observability"
)

func TestLoggerFromContext_ReturnsStoredLogger(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)

	ctx := observability.WithLogger(context.Background(), logger)

	got := observability.LoggerFromContext(ctx)

	if got != logger {
		t.Fatalf("expected the stored logger back, got a different instance")
	}
}

func TestLoggerFromContext_FallsBackToDefaultWhenAbsent(t *testing.T) {
	t.Parallel()

	got := observability.LoggerFromContext(context.Background())

	if got != slog.Default() {
		t.Fatalf("expected slog.Default() when no logger was stored, got a different instance")
	}
}
