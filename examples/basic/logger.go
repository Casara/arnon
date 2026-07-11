package main

import (
	"log/slog"
	"os"
)

// NewLogger creates the application logger.
//
// It writes structured JSON logs to stdout and sets itself as the
// slog default, so any package-level slog.Info/Debug/Error call in
// this example uses it without threading a logger through every
// function.
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
