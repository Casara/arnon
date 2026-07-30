package middleware

import (
	"fmt"
	"net"
	"net/netip"
)

// RealIPOption configures RealIP.
type RealIPOption func(*realIPConfig)

type realIPConfig struct {
	trustedProxies []netip.Prefix
	trustAll       bool
}

// WithTrustedProxies restricts RealIP to only believe the forwarding headers
// when the request arrives from one of the given networks - your load
// balancer, ingress controller or CDN egress ranges.
//
// This is the difference between a working rate limit and one that can be
// walked past. Without it, RealIP takes X-Forwarded-For from anyone, so a
// caller reaching the server directly can forge a new client identity on every
// request. With it, a request whose RemoteAddr is outside these networks is
// keyed on RemoteAddr itself, which cannot be forged over TCP.
//
// Each entry is a CIDR block; a bare address is accepted and treated as a
// single host (/32 or /128):
//
//	middleware.RealIP(middleware.WithTrustedProxies(
//		"10.0.0.0/8",       // internal load balancer
//		"192.168.1.7",      // a specific reverse proxy
//	))
//
// Panics on a malformed entry. Trusted proxy ranges are deployment
// configuration read at startup, so a typo is a programming error rather than
// something to discover from a wrong rate-limit key in production.
func WithTrustedProxies(networks ...string) RealIPOption {
	prefixes := make([]netip.Prefix, 0, len(networks))

	for _, network := range networks {
		prefixes = append(prefixes, mustParseNetwork(network))
	}

	return func(config *realIPConfig) {
		config.trustedProxies = append(config.trustedProxies, prefixes...)
		config.trustAll = false
	}
}

// WithTrustAnyProxy keeps RealIP believing the forwarding headers on every
// request, whatever its source. This is the default, for compatibility with a
// deployment that is already behind a proxy overwriting those headers.
//
// It exists so the choice can be written down: a reviewer reading
// RealIP(WithTrustAnyProxy()) sees a decision, where a bare RealIP() only
// shows an absence. Never use it on a server reachable directly from the
// internet - see WithTrustedProxies.
func WithTrustAnyProxy() RealIPOption {
	return func(config *realIPConfig) {
		config.trustedProxies = nil
		config.trustAll = true
	}
}

func mustParseNetwork(network string) netip.Prefix {
	prefix, err := netip.ParsePrefix(network)
	if err == nil {
		return prefix
	}

	address, addressErr := netip.ParseAddr(network)
	if addressErr != nil {
		panic(fmt.Sprintf(
			"middleware: WithTrustedProxies: %q is neither a CIDR block nor an address: %v",
			network, err,
		))
	}

	return netip.PrefixFrom(address, address.BitLen())
}

// trusts reports whether the forwarding headers on a request from remoteAddr
// may be believed.
func (config realIPConfig) trusts(remoteAddr string) bool {
	if config.trustAll {
		return true
	}

	if len(config.trustedProxies) == 0 {
		return true
	}

	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	address, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}

	// An IPv4 peer arriving over a dual-stack listener shows up as
	// ::ffff:a.b.c.d, which no plain IPv4 prefix would match.
	if address.Is4In6() {
		address = address.Unmap()
	}

	for _, prefix := range config.trustedProxies {
		if prefix.Contains(address) {
			return true
		}
	}

	return false
}
