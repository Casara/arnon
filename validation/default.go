package validation

import (
	"fmt"
	"sync"
)

//nolint:gochecknoglobals // Shared validator instance reuses validator caches.
var defaultValidator = sync.OnceValue(func() Validator {
	val, err := New()
	if err != nil {
		panic(fmt.Errorf("create default validator: %w", err))
	}

	return val
})

// Default returns the framework default validator instance.
//
// The validator is lazily initialized on first use and reused for the
// lifetime of the application. Sharing a single instance allows the
// underlying validator implementation to reuse its internal caches.
//
// Any custom rule registered via RegisterCustomRule before first use
// is applied automatically. Construction only fails when a rule is
// misconfigured (e.g. a reserved tag name), which is a startup-time
// programming error, hence the panic rather than a silently degraded
// validator.
//
// It should be sufficient for most use cases. Create a dedicated
// validator only when custom validation behavior is required.
func Default() Validator {
	return defaultValidator()
}
