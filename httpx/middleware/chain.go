package middleware

import (
	"compress/gzip"
	"fmt"
	"log/slog"
	"time"

	"github.com/casara/arnon/httpx/routing"
)

// CompressConfig configures the Compress middleware for use through
// ChainConfig/BuildChain, which - unlike Compress itself - takes
// configuration as a struct rather than positional parameters, for
// consistency with every other *Config type in this package.
type CompressConfig struct {
	// Level is the gzip compression level, as defined by
	// compress/gzip. Zero defaults to gzip.DefaultCompression.
	Level int

	// Types restricts compression to these response Content-Type
	// values (a trailing "/*" matches any subtype). Empty uses
	// Compress's own default list.
	Types []string
}

// ChainAnchor names a position in BuildChain's stage sequence, used
// by ChainConfig.Extra to say where a custom middleware belongs
// relative to arnon's own built-ins - see BuildChain's doc comment
// for the full sequence.
//
// An anchor names a position, not a specific middleware's presence:
// AnchorETag marks where ETag would run whether or not
// ChainConfig.ETag is actually true, so a custom middleware anchored
// there still gets a well-defined, predictable spot even when ETag
// itself is left out of that particular chain.
type ChainAnchor string

// The full set of anchors BuildChain recognizes, in chain order.
const (
	AnchorRecover         ChainAnchor = "recover"
	AnchorTimeout         ChainAnchor = "timeout"
	AnchorStripSlashes    ChainAnchor = "strip_slashes"
	AnchorRedirectSlashes ChainAnchor = "redirect_slashes"
	AnchorRealIP          ChainAnchor = "real_ip"
	AnchorRequestID       ChainAnchor = "request_id"
	AnchorSecureHeaders   ChainAnchor = "secure_headers"
	AnchorRateLimit       ChainAnchor = "rate_limit"
	AnchorThrottle        ChainAnchor = "throttle"
	AnchorETag            ChainAnchor = "etag"
	AnchorCompress        ChainAnchor = "compress"
	AnchorCORS            ChainAnchor = "cors"
	AnchorServiceDesc     ChainAnchor = "service_desc"
	AnchorLogging         ChainAnchor = "logging"
)

//nolint:gochecknoglobals // read-only reference data, not per-request state
var validChainAnchors = map[ChainAnchor]struct{}{
	AnchorRecover:         {},
	AnchorTimeout:         {},
	AnchorStripSlashes:    {},
	AnchorRedirectSlashes: {},
	AnchorRealIP:          {},
	AnchorRequestID:       {},
	AnchorSecureHeaders:   {},
	AnchorRateLimit:       {},
	AnchorThrottle:        {},
	AnchorETag:            {},
	AnchorCompress:        {},
	AnchorCORS:            {},
	AnchorServiceDesc:     {},
	AnchorLogging:         {},
}

// ExtraMiddleware inserts Middleware immediately before or after a
// named ChainAnchor in BuildChain's sequence - the escape hatch for a
// custom or third-party middleware with an ordering requirement
// against one of arnon's own built-ins (e.g. "must run after
// RateLimit, before ETag") that plain ChainConfig fields can't
// express on their own.
//
// Exactly one of Before/After must be set; BuildChain panics
// otherwise, and panics too if the anchor isn't one of the
// AnchorXxx constants (a typo here would otherwise silently drop the
// middleware from the chain instead of failing loudly).
type ExtraMiddleware struct {
	Middleware routing.Middleware
	Before     ChainAnchor
	After      ChainAnchor
}

func (extra ExtraMiddleware) anchor() ChainAnchor {
	if extra.Before != "" {
		return extra.Before
	}

	return extra.After
}

func validateExtraMiddleware(extra ExtraMiddleware) {
	if extra.Before == "" && extra.After == "" {
		panic("middleware: ChainConfig.Extra entry must set Before or After")
	}

	if extra.Before != "" && extra.After != "" {
		panic(fmt.Sprintf(
			"middleware: ChainConfig.Extra entry must set only one of Before/After, got Before=%q and After=%q",
			extra.Before,
			extra.After,
		))
	}

	if _, ok := validChainAnchors[extra.anchor()]; !ok {
		panic(fmt.Sprintf(
			"middleware: ChainConfig.Extra references unknown anchor %q",
			extra.anchor(),
		))
	}
}

// ChainConfig selects and configures ArNon's recommended global
// middleware chain (see BuildChain). A nil/zero field means that
// middleware is left out of the chain entirely, not "disabled" -
// there is no runtime cost or behavior difference from never having
// mentioned it.
//
// This only covers middleware meant to be global (wrapping the whole
// mux via Router.Use): AllowContentType, MaxBodyBytes and NoCache are
// deliberately not here, because they only make sense scoped to a
// specific group (e.g. an API's JSON-only route group), not the
// static /openapi.json or /docs endpoints - install those with
// Group.Use instead, same as before.
type ChainConfig struct {
	// Recover recovers panics into a Problem Details 500 when true.
	Recover bool

	// Timeout, when non-zero, aborts a request that runs longer than
	// this duration with a Problem Details 503.
	Timeout time.Duration

	// StripSlashes normalizes a trailing slash in place (no redirect)
	// when true. Mutually exclusive with RedirectSlashes - BuildChain
	// panics if both are set, since installing both means the second
	// one registered never actually has a trailing slash left to act
	// on.
	StripSlashes bool

	// RedirectSlashes 308-redirects a trailing slash instead of
	// normalizing it in place. Mutually exclusive with StripSlashes.
	RedirectSlashes bool

	// RealIP populates the client IP in the request context when true.
	RealIP bool

	// RequestID generates/propagates X-Request-Id when true.
	RequestID bool

	// SecureHeaders sets baseline security response headers when
	// non-nil.
	SecureHeaders *SecureHeadersConfig

	// RateLimit enables per-client rate limiting when non-nil.
	RateLimit *RateLimitConfig

	// Throttle enables a concurrent-request cap when non-nil.
	Throttle *ThrottleConfig

	// ETag enables conditional GET support when true.
	ETag bool

	// Compress enables gzip compression when non-nil.
	Compress *CompressConfig

	// CORS enables CORS handling when non-nil.
	CORS *CORSConfig

	// ServiceDescPath, when non-empty, adds a
	// Link: rel="service-desc" response header pointing at this path
	// (typically the OpenAPI document, e.g. "/openapi.json").
	ServiceDescPath string

	// Logger enables structured request logging when non-nil.
	Logger *slog.Logger

	// Extra inserts custom middleware at named positions relative to
	// the fields above - see ExtraMiddleware.
	Extra []ExtraMiddleware
}

// BuildChain returns arnon's recommended global middleware chain, in this
// order:
//
//	Recover, Timeout, StripSlashes/RedirectSlashes, RealIP, RequestID,
//	SecureHeaders, RateLimit, Throttle, ETag, Compress, CORS,
//	ServiceDesc, Logging
//
// The slice this returns is always internally consistent regardless
// of which fields are set, because the order is decided here once
// rather than left to every caller to get right on their own.
//
// This is the safe default for the common case. Router.Use still
// accepts any hand-built, fully custom []routing.Middleware for
// advanced setups (e.g. different rate limits per route group) - this
// function does not restrict that, it just means most callers never
// have to reason about ordering constraints (RequestID before
// Logging, ETag before Compress, Recover outermost, ...) themselves.
// A custom middleware that needs a specific spot relative to one of
// the built-ins can use ChainConfig.Extra instead of splitting the
// call in two.
func BuildChain(config ChainConfig) []routing.Middleware {
	if config.StripSlashes && config.RedirectSlashes {
		panic("middleware: ChainConfig.StripSlashes and RedirectSlashes are mutually exclusive")
	}

	for _, extra := range config.Extra {
		validateExtraMiddleware(extra)
	}

	chain := make([]routing.Middleware, 0, chainCapacityHint+len(config.Extra))

	chain = appendStage(chain, AnchorRecover, config.Recover, config.Extra, Recover)
	chain = appendStage(
		chain, AnchorTimeout, config.Timeout > 0, config.Extra,
		func() routing.Middleware { return Timeout(config.Timeout) },
	)
	chain = appendStage(chain, AnchorStripSlashes, config.StripSlashes, config.Extra, StripSlashes)
	chain = appendStage(
		chain, AnchorRedirectSlashes, config.RedirectSlashes, config.Extra, RedirectSlashes,
	)
	chain = appendStage(chain, AnchorRealIP, config.RealIP, config.Extra, RealIP)
	chain = appendStage(chain, AnchorRequestID, config.RequestID, config.Extra, RequestID)
	chain = appendStage(
		chain, AnchorSecureHeaders, config.SecureHeaders != nil, config.Extra,
		func() routing.Middleware { return SecureHeaders(*config.SecureHeaders) },
	)
	chain = appendStage(
		chain, AnchorRateLimit, config.RateLimit != nil, config.Extra,
		func() routing.Middleware { return RateLimit(*config.RateLimit) },
	)
	chain = appendStage(
		chain, AnchorThrottle, config.Throttle != nil, config.Extra,
		func() routing.Middleware { return Throttle(*config.Throttle) },
	)
	chain = appendStage(chain, AnchorETag, config.ETag, config.Extra, ETag)
	chain = appendStage(
		chain, AnchorCompress, config.Compress != nil, config.Extra,
		func() routing.Middleware { return newConfiguredCompress(config.Compress) },
	)
	chain = appendStage(
		chain, AnchorCORS, config.CORS != nil, config.Extra,
		func() routing.Middleware { return CORS(*config.CORS) },
	)
	chain = appendStage(
		chain, AnchorServiceDesc, config.ServiceDescPath != "", config.Extra,
		func() routing.Middleware { return ServiceDesc(config.ServiceDescPath) },
	)
	chain = appendStage(
		chain, AnchorLogging, config.Logger != nil, config.Extra,
		func() routing.Middleware { return Logging(config.Logger) },
	)

	return chain
}

func newConfiguredCompress(config *CompressConfig) routing.Middleware {
	level := config.Level
	if level == 0 {
		level = gzip.DefaultCompression
	}

	return Compress(level, config.Types...)
}

// appendStage appends, in order: any extra entries anchored Before
// anchor, build() if enabled, then any extra entries anchored After
// anchor. build is only called when enabled, so it can safely
// dereference config fields (e.g. *config.RateLimit) that are only
// valid when that stage is actually turned on.
func appendStage(
	chain []routing.Middleware,
	anchor ChainAnchor,
	enabled bool,
	extra []ExtraMiddleware,
	build func() routing.Middleware,
) []routing.Middleware {
	for _, item := range extra {
		if item.Before == anchor {
			chain = append(chain, item.Middleware)
		}
	}

	if enabled {
		chain = append(chain, build())
	}

	for _, item := range extra {
		if item.After == anchor {
			chain = append(chain, item.Middleware)
		}
	}

	return chain
}

// chainCapacityHint is the number of middlewares BuildChain installs
// when every ChainConfig field is set - just a starting capacity for
// the slice, not a hard limit.
const chainCapacityHint = 13
