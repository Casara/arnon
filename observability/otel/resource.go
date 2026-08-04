package otel

import (
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// newResource creates the OpenTelemetry resource shared by all telemetry
// signals.
func newResource(
	config Config,
) *sdkresource.Resource {
	return sdkresource.NewWithAttributes(
		semconv.SchemaURL,

		semconv.ServiceName(
			config.ServiceName,
		),

		semconv.ServiceVersion(
			config.ServiceVersion,
		),

		semconv.DeploymentEnvironmentName(
			config.Environment,
		),
	)
}
