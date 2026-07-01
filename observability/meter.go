package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

const meterName = "github.com/Casara/arnon"

func meter() metric.Meter {
	return otel.Meter(
		meterName,
	)
}
