// Package validation checks a bound request against its validate struct tags.
//
// Default returns the shared validator that httpx.Endpoint installs when
// EndpointConfig.Validator is nil; it is what most code uses without naming
// this package at all. Validate reports every failure in one pass, as a
// []problem.ValidationError, each one addressing the offending field with an
// RFC 6901 JSON Pointer built from its json tag - not from the Go field name,
// so the pointer matches what the client actually sent.
//
// Like binding.Decode, Validate returns a slice rather than an error: for
// request validation, stopping at the first failure is the wrong behavior.
//
// # Custom rules
//
// RegisterCustomRule is the only way to add a tag. One registration feeds all
// three surfaces that have to agree on what the tag means:
//
//	Func            the runtime check
//	Code, Message   the resulting problem.ValidationError
//	Schema          the generated OpenAPI schema for fields using the tag
//
// Registering the same thing in three places, or configuring the underlying
// validator directly, is how those three drift apart - which is why there is
// no second mechanism for it.
//
// Call it during bootstrap, before the first Endpoint that uses the tag is
// constructed.
//
// # Implementation
//
// The Validator interface is the seam; PlaygroundValidator is the
// implementation, built on github.com/go-playground/validator/v10. A custom
// rule is written against this package's own FieldContext, so nothing you
// write names that library.
//
// The one exception is WithConfigure, which hands you the underlying
// *validator.Validate on purpose - the escape hatch for something this
// package does not wrap. Using it ties that code to validator/v10, and a rule
// registered through it is invisible to OpenAPI schema generation, which is
// exactly what RegisterCustomRule exists to avoid.
package validation
