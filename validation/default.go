package validation

import "sync"

//nolint:gochecknoglobals // Shared validator instance reuses validator caches.
var defaultValidator = sync.OnceValue(func() Validator {
	return New()
})

// Default returns the framework default validator instance.
//
// The validator is lazily initialized on first use and reused for the
// lifetime of the application. Sharing a single instance allows the
// underlying validator implementation to reuse its internal caches.
//
// It should be sufficient for most use cases. Create a dedicated
// validator only when custom validation behavior is required.
func Default() Validator {
	return defaultValidator()
}
