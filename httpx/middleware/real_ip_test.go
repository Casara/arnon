package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casara/arnon/httpx/middleware"
)

func TestRealIP(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		forwarded    string
		forwardedFor string
		realIPHeader string
		remoteAddr   string
		wantRealIP   string
	}{
		{
			name:       "Forwarded with a plain IPv4 for=",
			forwarded:  "for=192.0.2.60;proto=http;by=203.0.113.43",
			wantRealIP: "192.0.2.60",
		},
		{
			name:       "Forwarded with a quoted bracketed IPv6 for=",
			forwarded:  `for="[2001:db8:cafe::17]:4711";proto=http`,
			wantRealIP: "2001:db8:cafe::17",
		},
		{
			name:       "Forwarded with multiple hops keeps only the first",
			forwarded:  "for=192.0.2.43, for=198.51.100.17",
			wantRealIP: "192.0.2.43",
		},
		{
			name:         "Forwarded takes precedence over X-Forwarded-For",
			forwarded:    "for=192.0.2.60",
			forwardedFor: "70.41.3.18",
			wantRealIP:   "192.0.2.60",
		},
		{
			// "unknown" means the sender doesn't know the client's
			// identity (RFC 7239 §7.1) - falls through just like an
			// absent header would.
			name:         "Forwarded for=unknown falls through to X-Forwarded-For",
			forwarded:    "for=unknown",
			forwardedFor: "70.41.3.18",
			wantRealIP:   "70.41.3.18",
		},
		{
			// An obfuscated identifier (RFC 7239 §6.3) isn't an IP, but
			// is still a stable per-client token worth keeping instead
			// of discarding it in favor of a weaker source.
			name:       "Forwarded with an obfuscated identifier is kept as-is",
			forwarded:  "for=_hidden",
			wantRealIP: "_hidden",
		},
		{
			name:       "Forwarded with no for= parameter falls through",
			forwarded:  "proto=http;by=203.0.113.43",
			remoteAddr: "192.0.2.1:54321",
			wantRealIP: "192.0.2.1",
		},
		{
			name:         "single X-Forwarded-For value",
			forwardedFor: "203.0.113.9",
			wantRealIP:   "203.0.113.9",
		},
		{
			// The common shape behind more than one proxy hop (e.g. a
			// CDN in front of a reverse proxy/ingress): each hop
			// appends its own peer address, leftmost is the original
			// client.
			name:         "multi-hop X-Forwarded-For keeps only the first entry",
			forwardedFor: "203.0.113.9, 70.41.3.18, 150.172.238.178",
			wantRealIP:   "203.0.113.9",
		},
		{
			name:         "multi-hop X-Forwarded-For trims surrounding whitespace",
			forwardedFor: "203.0.113.9 , 70.41.3.18",
			wantRealIP:   "203.0.113.9",
		},
		{
			name:         "falls back to X-Real-IP when X-Forwarded-For is absent",
			realIPHeader: "198.51.100.7",
			wantRealIP:   "198.51.100.7",
		},
		{
			name:       "falls back to RemoteAddr when no header is present",
			remoteAddr: "192.0.2.1:54321",
			wantRealIP: "192.0.2.1",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var gotRealIP string

			handler := middleware.RealIP()(http.HandlerFunc(func(
				_ http.ResponseWriter,
				request *http.Request,
			) {
				gotRealIP = middleware.RealIPFromContext(request.Context())
			}))

			request := httptest.NewRequest(http.MethodGet, "/", nil)

			if testCase.forwarded != "" {
				request.Header.Set("Forwarded", testCase.forwarded)
			}

			if testCase.forwardedFor != "" {
				request.Header.Set("X-Forwarded-For", testCase.forwardedFor)
			}

			if testCase.realIPHeader != "" {
				request.Header.Set("X-Real-IP", testCase.realIPHeader)
			}

			if testCase.remoteAddr != "" {
				request.RemoteAddr = testCase.remoteAddr
			}

			handler.ServeHTTP(httptest.NewRecorder(), request)

			if gotRealIP != testCase.wantRealIP {
				t.Errorf("expected real IP %q, got %q", testCase.wantRealIP, gotRealIP)
			}
		})
	}
}
