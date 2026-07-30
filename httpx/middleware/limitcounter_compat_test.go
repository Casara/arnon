package middleware_test

import (
	"time"

	"github.com/casara/arnon/httpx/middleware"
)

// LimitCounter deliberately mirrors github.com/go-chi/httprate's interface of
// the same name, method for method, so that a backend written for either
// project works with the other unchanged - go-chi/httprate-redis, for
// instance, satisfies this one with no adapter at all.
//
// Go interfaces are structural, so that compatibility is not something either
// side declares: it holds only as long as the method sets stay identical.
// httprateLimitCounter below is a copy of httprate's interface as of v0.16.0,
// written out rather than imported, because importing httprate to test this
// would add a dependency to a project that deliberately has none for this.
//
// If a change here breaks the assertion below, that is the signal: it silently
// cuts arnon off from every existing httprate backend. Either revert it, or
// accept the break knowingly and update this file and the docs that promise
// the compatibility.
type httprateLimitCounter interface {
	Config(requestLimit int, windowLength time.Duration)
	Increment(key string, currentWindow time.Time) error
	IncrementBy(key string, currentWindow time.Time, amount int) error
	Get(key string, currentWindow, previousWindow time.Time) (int, int, error)
}

// Compile-time assertions, both directions: any httprate backend satisfies
// arnon's interface, and any arnon backend satisfies httprate's.
var (
	_ interface {
		httprateLimitCounter
	} = (middleware.LimitCounter)(nil)

	_ interface {
		middleware.LimitCounter
	} = (httprateLimitCounter)(nil)
)
