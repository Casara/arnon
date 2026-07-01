package otel

import "time"

// DefaultShutdownTimeout defines how long the framework waits for
// observability providers to flush pending telemetry during shutdown.
const DefaultShutdownTimeout = 10 * time.Second

// Config configures OpenTelemetry.
type Config struct {
	// ServiceName identifies the service in telemetry backends.
	ServiceName string

	// ServiceVersion identifies the service version.
	ServiceVersion string

	// Environment identifies the deployment environment.
	// Examples: development, staging, production.
	Environment string

	// MetricsEnabled enables OpenTelemetry metrics.
	MetricsEnabled bool

	// TracesEnabled enables OpenTelemetry tracing.
	TracesEnabled bool

	// MetricOTLPEndpoint defines the OTLP endpoint used for metrics.
	MetricOTLPEndpoint string

	// TraceOTLPEndpoint defines the OTLP endpoint used for traces.
	//
	// Examples:
	// localhost:4317
	// localhost:4318
	// otel-collector:4317
	TraceOTLPEndpoint string

	// OTLPInsecure disables TLS when connecting to the collector.
	OTLPInsecure bool

	// ShutdownTimeout defines the maximum time allowed for graceful shutdown.
	ShutdownTimeout time.Duration
}

func (config Config) withDefaults() Config {
	if config.Environment == "" {
		config.Environment = "development"
	}

	if config.ShutdownTimeout == 0 {
		config.ShutdownTimeout = DefaultShutdownTimeout
	}

	return config
}
