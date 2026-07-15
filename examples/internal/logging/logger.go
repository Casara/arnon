// Package logging provides the shared logger setup used by every
// runnable example under examples/cmd, so it isn't duplicated in each
// one.
package logging

import (
	"log/slog"
	"os"
)

// NewLogger creates the application logger.
//
// It writes structured JSON logs to stdout and sets itself as the
// slog default, so any package-level slog.Info/Debug/Error call in an
// example uses it without threading a logger through every function.
func NewLogger() *slog.Logger {
	logger := slog.New(
		slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: slog.LevelDebug,
			},
		),
	)

	slog.SetDefault(logger)

	return logger
}
