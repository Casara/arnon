package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/Casara/arnon/httpx/routing"
)

// RealIP extracts client IP.
//
// Sources are tried in order, falling through whenever one is absent:
//
//  1. Forwarded (RFC 7239) - the standardized replacement for the two
//     headers below, so it takes precedence when present.
//  2. X-Forwarded-For - by convention, a comma-separated list, one
//     entry per proxy hop, leftmost being the original client, so
//     only the first entry is used. Without this, a request that
//     passed through more than one proxy (common behind a CDN in
//     front of a reverse proxy/ingress) would set the "IP" to the
//     entire comma-separated string, silently breaking anything
//     downstream that expects a real IP, such as RateLimit's
//     CanonicalizeIP.
//  3. X-Real-IP.
//  4. request.RemoteAddr.
func RealIP() routing.Middleware {
	return func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			ip := parseForwardedFor(
				request.Header.Get("Forwarded"),
			)

			if ip == "" {
				ip = firstForwardedFor(
					request.Header.Get("X-Forwarded-For"),
				)
			}

			if ip == "" {
				ip = request.Header.Get(
					"X-Real-IP",
				)
			}

			if ip == "" {
				host, _, err := net.SplitHostPort(
					request.RemoteAddr,
				)
				if err == nil {
					ip = host
				}
			}

			ip = strings.TrimSpace(ip)

			ctx := context.WithValue(
				request.Context(),
				realIPContextKey,
				ip,
			)

			next.ServeHTTP(
				writer,
				request.WithContext(ctx),
			)
		})
	}
}

// firstForwardedFor returns the first, leftmost entry of a
// X-Forwarded-For header value, trimmed of surrounding whitespace.
// Returns "" unchanged when header is already empty.
func firstForwardedFor(header string) string {
	if header == "" {
		return ""
	}

	if firstComma := strings.IndexByte(header, ','); firstComma != -1 {
		header = header[:firstComma]
	}

	return strings.TrimSpace(header)
}

// parseForwardedFor extracts the client identifier (the "for"
// parameter) from the first hop of a Forwarded header (RFC 7239),
// which standardizes and replaces the de facto X-Forwarded-For/
// X-Real-IP headers used elsewhere in this file.
//
// Returns "" - so callers fall through to the next source - when the
// header is empty, has no "for" parameter on its first hop, or that
// parameter is the literal "unknown" (RFC 7239 §7.1: the sender
// doesn't know the client's identity).
//
// An obfuscated identifier (a "for" value starting with "_", RFC 7239
// §6.3, letting an intermediary avoid revealing a real address) is
// returned as-is: it's still a stable per-client token, usable as a
// rate-limit key even though it isn't an IP address.
func parseForwardedFor(header string) string {
	if header == "" {
		return ""
	}

	firstHop := splitTopLevel(header, ',')[0]

	for _, pair := range splitTopLevel(firstHop, ';') {
		name, value, found := strings.Cut(pair, "=")
		if !found {
			continue
		}

		if !strings.EqualFold(strings.TrimSpace(name), "for") {
			continue
		}

		value = unquoteForwardedValue(strings.TrimSpace(value))

		if value == "" || strings.EqualFold(value, "unknown") {
			return ""
		}

		return stripForwardedNodePort(value)
	}

	return ""
}

// splitTopLevel splits s on sep, ignoring any sep byte that appears
// inside a double-quoted substring - Forwarded parameter values are
// quoted-strings exactly when they contain characters (like the ":"
// and "]" of a bracketed IPv6 address) that would otherwise be
// ambiguous with the "," (hop) and ";" (parameter) separators.
func splitTopLevel(text string, sep byte) []string {
	parts := make([]string, 0, 1)

	inQuotes := false
	start := 0

	for i := range len(text) {
		switch text[i] {
		case '"':
			inQuotes = !inQuotes
		case sep:
			if !inQuotes {
				parts = append(parts, text[start:i])

				start = i + 1
			}
		}
	}

	return append(parts, text[start:])
}

// unquoteForwardedValue strips the surrounding double quotes of a
// Forwarded parameter value, if present, and undoes backslash
// escaping of quotes/backslashes - the only two characters RFC 7230's
// quoted-string grammar (which RFC 7239 reuses) allows to be escaped.
func unquoteForwardedValue(value string) string {
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return value
	}

	return forwardedQuotedPairReplacer.Replace(value[1 : len(value)-1])
}

//nolint:gochecknoglobals // read-only, immutable *strings.Replacer built once
var forwardedQuotedPairReplacer = strings.NewReplacer(`\"`, `"`, `\\`, `\`)

// stripForwardedNodePort removes the optional ":" node-port suffix
// from a Forwarded "for" value. Per RFC 7239 §6, an IPv6 nodename is
// always bracketed (with or without a following port), so any value
// starting with "[" has its address extracted from between the
// brackets; anything else (IPv4, or an obfuscated token, neither of
// which contain "[") is split on its first ":", if any.
func stripForwardedNodePort(node string) string {
	if strings.HasPrefix(node, "[") {
		if end := strings.IndexByte(node, ']'); end != -1 {
			return node[1:end]
		}

		return node
	}

	if host, _, found := strings.Cut(node, ":"); found {
		return host
	}

	return node
}
