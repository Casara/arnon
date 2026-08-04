// Package otel provides OpenTelemetry integrations for arnon.
//
// The package contains adapters, middleware and helpers that integrate
// arnon with the OpenTelemetry ecosystem.
//
// OpenTelemetry-specific APIs and types should remain contained within
// this package whenever possible. Other packages should depend on the
// abstractions exposed by the observability package instead of directly using
// OpenTelemetry APIs.
package otel
