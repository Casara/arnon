// Package otel provides OpenTelemetry integrations for the foundation.
//
// The package contains adapters, middleware and helpers that integrate
// the foundation with the OpenTelemetry ecosystem.
//
// OpenTelemetry-specific APIs and types should remain contained within
// this package whenever possible. Other packages should depend on the
// abstractions exposed by pkg/observability instead of directly using
// OpenTelemetry APIs.
package otel
