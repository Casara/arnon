package middleware_test

import (
	"bytes"
	"compress/gzip"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/casara/arnon/httpx/middleware"
	"github.com/casara/arnon/httpx/routing"
)

func TestBuildChain_EmptyConfigProducesEmptyChain(t *testing.T) {
	t.Parallel()

	chain := middleware.BuildChain(middleware.ChainConfig{})

	if len(chain) != 0 {
		t.Errorf("expected an empty chain, got %d middlewares", len(chain))
	}
}

func TestBuildChain_PanicsWhenStripAndRedirectSlashesBothSet(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Error(
				"expected BuildChain to panic when StripSlashes and RedirectSlashes are both set",
			)
		}
	}()

	middleware.BuildChain(middleware.ChainConfig{
		StripSlashes:    true,
		RedirectSlashes: true,
	})
}

// TestBuildChain_RequestIDRunsBeforeLogging proves the order BuildChain
// documents is the order it actually produces: RequestID populates the
// context Logging reads from, so a chain built from a config with both
// enabled must show request_id in the log output - if BuildChain ever
// regressed to installing Logging first, this would fail.
func TestBuildChain_RequestIDRunsBeforeLogging(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	router := routing.NewRouter()
	router.Use(middleware.BuildChain(middleware.ChainConfig{
		RequestID: true,
		Logger:    logger,
	})...)

	router.GET("/ping", http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusOK)
	}))

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))

	if !strings.Contains(logBuffer.String(), `"request_id"`) {
		t.Errorf("expected the log line to contain request_id, got %q", logBuffer.String())
	}
}

// TestBuildChain_ETagRunsBeforeCompress proves ETag sees the
// already-compressed bytes: the ETag for the same logical response
// must differ between a plain request and one that accepts gzip, since
// they hash different bytes on the wire. If BuildChain ever regressed
// to installing Compress before ETag, both requests would hash the
// same (uncompressed) bytes and produce identical ETags.
func TestBuildChain_ETagRunsBeforeCompress(t *testing.T) {
	t.Parallel()

	router := routing.NewRouter()
	router.Use(middleware.BuildChain(middleware.ChainConfig{
		ETag: true,
		Compress: &middleware.CompressConfig{
			Level: gzip.DefaultCompression,
		},
	})...)

	router.GET("/body", http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"id":"usr_123"}`))
	}))

	plain := httptest.NewRecorder()
	router.ServeHTTP(plain, httptest.NewRequest(http.MethodGet, "/body", nil))

	gzipped := httptest.NewRecorder()
	gzipRequest := httptest.NewRequest(http.MethodGet, "/body", nil)
	gzipRequest.Header.Set("Accept-Encoding", "gzip")
	router.ServeHTTP(gzipped, gzipRequest)

	if gzipped.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf(
			"expected the gzip request to be compressed, got Content-Encoding %q",
			gzipped.Header().Get("Content-Encoding"),
		)
	}

	plainETag := plain.Header().Get("ETag")
	gzipETag := gzipped.Header().Get("ETag")

	if plainETag == "" || gzipETag == "" {
		t.Fatalf("expected both responses to carry an ETag, got %q and %q", plainETag, gzipETag)
	}

	if plainETag == gzipETag {
		t.Error("expected different ETags for the compressed and uncompressed representations")
	}
}

// TestBuildChain_RecoverIsOutermost proves a panic raised deep inside
// the chain (from the actual handler, past every other configured
// middleware) is still turned into a Problem Details 500 rather than
// crashing the process - only possible if Recover wraps everything
// else BuildChain installs.
func TestBuildChain_RecoverIsOutermost(t *testing.T) {
	t.Parallel()

	router := routing.NewRouter()
	router.Use(middleware.BuildChain(middleware.ChainConfig{
		Recover:       true,
		RealIP:        true,
		RequestID:     true,
		SecureHeaders: &middleware.SecureHeadersConfig{},
		RateLimit: &middleware.RateLimitConfig{
			RequestLimit: 100,
			WindowLength: time.Minute,
		},
	})...)

	router.GET("/boom", http.HandlerFunc(func(
		http.ResponseWriter,
		*http.Request,
	) {
		panic("boom")
	}))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/boom", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
}

// TestBuildChain_SecureHeadersAppliesEvenWhenRateLimited proves
// SecureHeaders sits early enough to still land on a 429 produced by
// RateLimit further down the chain.
func TestBuildChain_SecureHeadersAppliesEvenWhenRateLimited(t *testing.T) {
	t.Parallel()

	router := routing.NewRouter()
	router.Use(middleware.BuildChain(middleware.ChainConfig{
		SecureHeaders: &middleware.SecureHeadersConfig{},
		RateLimit: &middleware.RateLimitConfig{
			RequestLimit: 1,
			WindowLength: time.Minute,
			KeyFunc: func(*http.Request) string {
				return "same-client"
			},
		},
	})...)

	router.GET("/ping", http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusOK)
	}))

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/ping", nil))

	limited := httptest.NewRecorder()
	router.ServeHTTP(limited, httptest.NewRequest(http.MethodGet, "/ping", nil))

	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, limited.Code)
	}

	if limited.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("expected SecureHeaders to apply even to a 429 response")
	}
}

// markerMiddleware appends name to order when it runs, then calls
// next - used to observe the actual execution order BuildChain
// produces, the same way router_test.go proves middleware ordering.
func markerMiddleware(name string, order *[]string) routing.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			*order = append(*order, name)

			next.ServeHTTP(writer, request)
		})
	}
}

func TestBuildChain_ExtraAfterAnchorRunsRightAfterThatStage(t *testing.T) {
	t.Parallel()

	var order []string

	router := routing.NewRouter()
	router.Use(middleware.BuildChain(middleware.ChainConfig{
		RealIP:    true,
		RequestID: true,
		Extra: []middleware.ExtraMiddleware{
			{
				Middleware: markerMiddleware("xpto", &order),
				After:      middleware.AnchorRequestID,
			},
		},
	})...)
	router.Use(markerMiddleware("logging-stand-in", &order))

	router.GET("/ping", http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))

	want := []string{"xpto", "logging-stand-in"}
	if len(order) != len(want) || order[0] != want[0] || order[1] != want[1] {
		t.Errorf("expected order %v, got %v", want, order)
	}
}

// TestBuildChain_ExtraBeforeAnchorPositioning covers two cases with
// the same shape: an Extra anchored Before a stage runs right before
// it when that stage is enabled, and an anchor names a *position*,
// not a specific middleware's presence - Extra anchored at ETag still
// lands in the right spot even when ChainConfig.ETag is false.
func TestBuildChain_ExtraBeforeAnchorPositioning(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		config func(order *[]string) middleware.ChainConfig
	}{
		{
			name: "stage enabled",
			config: func(order *[]string) middleware.ChainConfig {
				return middleware.ChainConfig{
					RealIP: true,
					Extra: []middleware.ExtraMiddleware{
						{
							Middleware: markerMiddleware("xpto", order),
							Before:     middleware.AnchorRealIP,
						},
					},
				}
			},
		},
		{
			name: "stage disabled - anchor still marks the position",
			config: func(order *[]string) middleware.ChainConfig {
				return middleware.ChainConfig{
					RealIP: true,
					// ETag deliberately left false.
					Extra: []middleware.ExtraMiddleware{
						{Middleware: markerMiddleware("xpto", order), After: middleware.AnchorETag},
					},
				}
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var order []string

			router := routing.NewRouter()
			router.Use(middleware.BuildChain(testCase.config(&order))...)

			router.GET("/ping", http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				order = append(order, "handler")

				writer.WriteHeader(http.StatusOK)
			}))

			router.ServeHTTP(
				httptest.NewRecorder(),
				httptest.NewRequest(http.MethodGet, "/ping", nil),
			)

			want := []string{"xpto", "handler"}
			if len(order) != len(want) || order[0] != want[0] || order[1] != want[1] {
				t.Errorf("expected order %v, got %v", want, order)
			}
		})
	}
}

func TestBuildChain_MultipleExtrasAtSameAnchorStackInSliceOrder(t *testing.T) {
	t.Parallel()

	var order []string

	chain := middleware.BuildChain(middleware.ChainConfig{
		Extra: []middleware.ExtraMiddleware{
			{Middleware: markerMiddleware("first", &order), After: middleware.AnchorRecover},
			{Middleware: markerMiddleware("second", &order), After: middleware.AnchorRecover},
		},
	})

	handler := routing.Chain(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusOK)
	}), chain...)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	want := []string{"first", "second"}
	if len(order) != len(want) || order[0] != want[0] || order[1] != want[1] {
		t.Errorf("expected order %v, got %v", want, order)
	}
}

func TestBuildChain_PanicsWhenExtraSetsNeitherBeforeNorAfter(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Error(
				"expected BuildChain to panic when an Extra entry sets neither Before nor After",
			)
		}
	}()

	middleware.BuildChain(middleware.ChainConfig{
		Extra: []middleware.ExtraMiddleware{
			{Middleware: markerMiddleware("xpto", &[]string{})},
		},
	})
}

func TestBuildChain_PanicsWhenExtraSetsBothBeforeAndAfter(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Error("expected BuildChain to panic when an Extra entry sets both Before and After")
		}
	}()

	middleware.BuildChain(middleware.ChainConfig{
		Extra: []middleware.ExtraMiddleware{
			{
				Middleware: markerMiddleware("xpto", &[]string{}),
				Before:     middleware.AnchorRateLimit,
				After:      middleware.AnchorETag,
			},
		},
	})
}

func TestBuildChain_PanicsWhenExtraReferencesUnknownAnchor(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Error("expected BuildChain to panic when an Extra entry references an unknown anchor")
		}
	}()

	middleware.BuildChain(middleware.ChainConfig{
		Extra: []middleware.ExtraMiddleware{
			{
				Middleware: markerMiddleware("xpto", &[]string{}),
				After:      middleware.ChainAnchor("not_a_real_anchor"),
			},
		},
	})
}
