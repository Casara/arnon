package routing

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var (
	ErrInvalidPattern = errors.New("routing: invalid route pattern, expected 'METHOD /path'")
	ErrInvalidMethod  = errors.New("routing: invalid HTTP method in route pattern")
	ErrInvalidPath    = errors.New(
		"routing: invalid path in route pattern, path must start with '/'",
	)
)

func joinPattern(prefix, pattern string) string {
	method, path, err := splitPattern(pattern)
	if err != nil {
		panic(err)
	}

	return method + " " + joinPath(
		prefix,
		path,
	)
}

func joinPath(left, right string) string {
	left = strings.TrimSuffix(
		left,
		"/",
	)

	right = strings.TrimPrefix(
		right,
		"/",
	)

	switch {
	case left == "":
		return "/" + right

	case right == "":
		return left

	default:
		return left + "/" + right
	}
}

func splitPattern(pattern string) (
	string, // method
	string, // path
	error, // err
) {
	parts := strings.Fields(pattern)

	if len(parts) != 2 { //nolint:mnd // string validation split into two parts
		return "", "", fmt.Errorf("pattern %q: %w", pattern, ErrInvalidPattern)
	}

	method := parts[0]
	path := parts[1]

	switch method {
	case
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions:

	default:
		return "", "", fmt.Errorf("method %q in pattern %q: %w", method, pattern, ErrInvalidMethod)
	}

	if !strings.HasPrefix(path, "/") {
		return "", "", fmt.Errorf("path %q in pattern %q: %w", path, pattern, ErrInvalidPath)
	}

	return method, path, nil
}
