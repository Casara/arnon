package sanitize

import "sync"

// Sanitizer functions are shared framework-wide metadata - which tag
// name maps to which transform - not per-instance state, which is why
// they live in a global registry rather than being threaded through
// every Endpoint call, mirroring validation's custom rule registry.
//
//nolint:gochecknoglobals // shared framework-wide tag metadata, see comment above
var (
	registryMu sync.RWMutex
	registry   = make(map[string]func(string) string)
	seedOnce   sync.Once
)

// RegisterFunc registers a named string transform globally, usable
// via the `sanitize:"tag"` struct tag on any string field reachable
// from a request DTO.
//
// It is intended to be called once during application bootstrap,
// before the first Endpoint using it is constructed - Prepare and
// Apply both resolve tag names against this same registry. Registering
// a tag that already exists - including a built-in one such as "trim"
// or "email" - replaces the previous definition.
func RegisterFunc(tag string, fn func(string) string) error {
	if tag == "" {
		return ErrSanitizerTagEmpty
	}

	if fn == nil {
		return ErrSanitizerFuncNil
	}

	seedBuiltins()

	registryMu.Lock()
	defer registryMu.Unlock()

	registry[tag] = fn

	return nil
}

// lookupFunc returns the sanitizer registered for tag, if any.
func lookupFunc(tag string) (func(string) string, bool) {
	seedBuiltins()

	registryMu.RLock()
	defer registryMu.RUnlock()

	fn, ok := registry[tag]

	return fn, ok
}

// seedBuiltins registers the framework's built-in sanitizers ("trim",
// "email") exactly once, lazily. gochecknoinits forbids doing this in
// an init func, so every entry point that reads or writes the
// registry calls this first instead - the same reason
// validation.Default() uses sync.OnceValue rather than an init func.
func seedBuiltins() {
	seedOnce.Do(func() {
		registryMu.Lock()
		defer registryMu.Unlock()

		registry["trim"] = trim
		registry["email"] = normalizeEmail
	})
}
