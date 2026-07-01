package otel

import (
	"fmt"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
)

// startRuntimeMetrics registers Go runtime metrics.
func startRuntimeMetrics() error {
	err := runtime.Start()
	if err != nil {
		return fmt.Errorf(
			"start runtime metrics: %w",
			err,
		)
	}

	return nil
}
