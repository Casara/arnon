package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casara/arnon/httpx/middleware"
)

// forwardedClientIP is what every request below claims to come from, so each
// test only varies the peer the request actually arrives from.
const forwardedClientIP = "1.2.3.4"

// resolveIP runs RealIP with options over a request arriving from remoteAddr
// and claiming forwardedClientIP, and reports the IP the middleware settled on.
func resolveIP(
	t *testing.T,
	remoteAddr string,
	options ...middleware.RealIPOption,
) string {
	t.Helper()

	var resolved string

	handler := middleware.RealIP(options...)(http.HandlerFunc(func(
		_ http.ResponseWriter,
		request *http.Request,
	) {
		resolved = middleware.RealIPFromContext(request.Context())
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = remoteAddr

	request.Header.Set("X-Forwarded-For", forwardedClientIP)

	handler.ServeHTTP(httptest.NewRecorder(), request)

	return resolved
}

// The whole point of the trusted-proxy list: a caller reaching the server
// directly cannot choose its own rate-limit key.
func TestRealIP_ForgedHeaderIgnoredFromUntrustedPeer(t *testing.T) {
	t.Parallel()

	resolved := resolveIP(
		t,
		"203.0.113.9:54321",
		middleware.WithTrustedProxies("10.0.0.0/8"),
	)

	if resolved != "203.0.113.9" {
		t.Errorf("forged X-Forwarded-For was believed: got %q", resolved)
	}
}

func TestRealIP_HeaderHonoredFromTrustedProxy(t *testing.T) {
	t.Parallel()

	resolved := resolveIP(
		t,
		"10.1.2.3:54321",
		middleware.WithTrustedProxies("10.0.0.0/8"),
	)

	if resolved != "1.2.3.4" {
		t.Errorf("header from a trusted proxy was ignored: got %q", resolved)
	}
}

// A bare address is accepted and means that host only.
func TestRealIP_SingleHostTrustEntry(t *testing.T) {
	t.Parallel()

	trusted := resolveIP(
		t,
		"192.168.1.7:1",
		middleware.WithTrustedProxies("192.168.1.7"))
	if trusted != "1.2.3.4" {
		t.Errorf("trusted host rejected: got %q", trusted)
	}

	neighbour := resolveIP(
		t,
		"192.168.1.8:1",
		middleware.WithTrustedProxies("192.168.1.7"))
	if neighbour != "192.168.1.8" {
		t.Errorf("neighbouring host was trusted: got %q", neighbour)
	}
}

// An IPv4 peer on a dual-stack listener arrives as ::ffff:a.b.c.d, which no
// plain IPv4 prefix matches unless it is unmapped first.
func TestRealIP_IPv4MappedPeerMatchesIPv4Prefix(t *testing.T) {
	t.Parallel()

	resolved := resolveIP(
		t,
		"[::ffff:10.1.2.3]:54321",
		middleware.WithTrustedProxies("10.0.0.0/8"))

	if resolved != "1.2.3.4" {
		t.Errorf("IPv4-mapped peer did not match the IPv4 prefix: got %q", resolved)
	}
}

// No options keeps the previous behavior, which is correct behind a proxy that
// overwrites the headers.
func TestRealIP_WithoutOptionsTrustsAnyPeer(t *testing.T) {
	t.Parallel()

	resolved := resolveIP(t, "203.0.113.9:54321")

	if resolved != "1.2.3.4" {
		t.Errorf("got %q", resolved)
	}
}

func TestRealIP_MalformedTrustedNetworkPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Error("a malformed network did not panic")
		}
	}()

	middleware.WithTrustedProxies("not-a-network")
}
