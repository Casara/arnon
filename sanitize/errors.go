package sanitize

import "errors"

// ErrSanitizerTagEmpty indicates that RegisterFunc was called without
// a tag name.
var ErrSanitizerTagEmpty = errors.New("sanitizer tag must not be empty")

// ErrSanitizerFuncNil indicates that RegisterFunc was called without
// a transform function.
var ErrSanitizerFuncNil = errors.New("sanitizer func must not be nil")

// ErrUnknownSanitizer indicates that a `sanitize` struct tag
// references a tag name with no function registered for it.
var ErrUnknownSanitizer = errors.New("unknown sanitizer")
