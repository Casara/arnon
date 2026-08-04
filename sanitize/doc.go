// Package sanitize transforms request data before validation runs -
// trimming whitespace, normalizing case, or any custom transform
// registered via RegisterFunc - so a validator like `required` or
// `min` sees the value a client actually intends, not raw bytes that
// happen to satisfy the check without meaning to (e.g. "C " passing
// `min=2` on its untrimmed length, even though the intended value is
// a single character).
//
// A `sanitize:"trim,lower"` struct tag chains named transforms, in
// order, resolved against the same registry RegisterFunc feeds. A
// struct field is always recursed into, mirroring how validator/v10
// dives into a nested struct automatically; a slice/array/map field
// needs its tag to start with "dive" to apply the remaining tokens to
// each element, mirroring validate's own convention.
//
// Anything that doesn't affect whether validation passes - display
// formatting, computed fields, masking sensitive data in a response -
// is out of scope here: that belongs in the handler, not in a
// sanitizer.
package sanitize
