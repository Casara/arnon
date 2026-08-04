// Package observability provides abstractions for distributed tracing,
// metrics and request correlation.
//
// The package intentionally avoids exposing vendor-specific APIs to the
// rest of arnon. This allows the underlying observability
// implementation to evolve without affecting application code.
//
// Subpackages provide integrations with specific technologies such as
// OpenTelemetry.
package observability
