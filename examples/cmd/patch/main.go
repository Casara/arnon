// Command patch demonstrates deriving PATCH from an existing GET+PUT
// pair via httpx/patch.From: RFC 7386 (JSON Merge Patch) and RFC 6902
// (JSON Patch) are both supported, selected by the incoming request's
// Content-Type, with no change to the GET/PUT handlers themselves.
//
// It also demonstrates the write-precondition mechanism
// (httpx/precondition) that closes patch.From's own optimistic-
// concurrency gap: the GET route is wrapped in middleware.ETag(), and
// store.put checks the incoming If-Match/If-Unmodified-Since against
// the profile's current state before applying an update - so both a
// direct PUT and a derived PATCH with a stale If-Match get 412
// Precondition Failed instead of silently overwriting a change they
// never saw.
//
// See httpx/patch's package doc comment for the mechanism (internal
// GET, apply the patch to the raw JSON, internal PUT).
package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/casara/arnon/httpx"
	"github.com/casara/arnon/httpx/middleware"
	"github.com/casara/arnon/httpx/patch"
	"github.com/casara/arnon/httpx/precondition"
	"github.com/casara/arnon/httpx/routing"
	"github.com/casara/arnon/openapi"
	"github.com/casara/arnon/problem"

	"github.com/casara/arnon/examples/internal/logging"
)

const readHeaderTimeout = 5 * time.Second

// Profile is the resource this example exposes at /profiles/{id} -
// deliberately small, just enough fields to demonstrate a merge-patch
// deletion (Bio, via `{"bio":null}`) and a JSON Patch array operation
// (Tags).
type Profile struct {
	Name string   `json:"name"`
	Bio  string   `json:"bio,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

// notFoundError is returned by store.get when id isn't in the store.
type notFoundError struct {
	id string
}

func (err notFoundError) Error() string {
	return fmt.Sprintf("profile %q not found", err.id)
}

// store is a tiny in-memory, mutex-protected profile database - real
// state (unlike examples/internal/users's stateless handlers) is what
// makes the derived PATCH demonstrable end to end: fetch the current
// value, apply a patch, put it back.
type store struct {
	mu       sync.Mutex
	profiles map[string]Profile
}

func newStore() *store {
	return &store{
		mu: sync.Mutex{},
		profiles: map[string]Profile{
			"ada": {
				Name: "Ada Lovelace",
				Bio:  "Mathematician and writer",
				Tags: []string{"mathematics"},
			},
		},
	}
}

// getProfileRequest is the request for store.get.
type getProfileRequest struct {
	ID string `path:"id"`
}

func (s *store) get(
	_ context.Context,
	request getProfileRequest,
) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, ok := s.profiles[request.ID]
	if !ok {
		return Profile{}, notFoundError{id: request.ID}
	}

	return profile, nil
}

// putProfileRequest is the request for store.put - a full
// replacement, same as any other PUT. IfMatch is bound the same way
// any other header is - httpx.HandlerFunc never receives
// *http.Request, so precondition.Check needs the value handed to it.
// Mixes single-purpose tags on purpose (ID/IfMatch alone, the rest
// json/validate alone), same as examples/internal/users.GetUserRequest.
//
//nolint:tagalign // see doc comment above
type putProfileRequest struct {
	ID      string   `path:"id"`
	IfMatch string   `          header:"If-Match"`
	Name    string   `                            json:"name" validate:"required"`
	Bio     string   `                            json:"bio"`
	Tags    []string `                            json:"tags"`
}

func (s *store) put(
	_ context.Context,
	request putProfileRequest,
) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, exists := s.profiles[request.ID]

	var currentETag string

	if exists {
		var err error

		currentETag, err = computeETag(current)
		if err != nil {
			return Profile{}, fmt.Errorf("compute current ETag: %w", err)
		}
	}

	problemInstance := precondition.Check(
		precondition.Headers{IfMatch: request.IfMatch},
		precondition.State{ETag: currentETag},
		precondition.Config{Require: false},
	)
	if problemInstance != nil {
		return Profile{}, problemInstance
	}

	profile := Profile{
		Name: request.Name,
		Bio:  request.Bio,
		Tags: request.Tags,
	}

	s.profiles[request.ID] = profile

	return profile, nil
}

// computeETag mirrors middleware.ETag's own algorithm (a strong ETag:
// FNV-1a 64-bit hash of the exact bytes httpx.WriteJSON would encode,
// hex-encoded and quoted) - not exported from httpx/middleware, so
// store.put duplicates the ~5 lines here to independently compute the
// current profile's ETag before checking preconditions against it.
func computeETag(profile Profile) (string, error) {
	buffer := &bytes.Buffer{}

	err := json.NewEncoder(buffer).Encode(profile)
	if err != nil {
		return "", fmt.Errorf("encode profile: %w", err)
	}

	hasher := fnv.New64a()
	_, _ = hasher.Write(buffer.Bytes())

	return `"` + hex.EncodeToString(hasher.Sum(nil)) + `"`, nil
}

func mapProfileError(err error) *problem.Problem {
	var notFound notFoundError
	if errors.As(err, &notFound) {
		return problem.NewNotFound(notFound.Error())
	}

	var problemInstance *problem.Problem
	if errors.As(err, &problemInstance) {
		return problemInstance
	}

	return problem.NewInternal("")
}

func main() {
	logging.NewLogger()

	profileStore := newStore()

	generator := openapi.NewGenerator(openapi.Info{
		Title:   "Example API",
		Version: "1.0.0",
	})

	router := routing.NewRouter(
		routing.WithOpenAPI(generator),
	)

	getHandler := httpx.Endpoint(profileStore.get, httpx.EndpointConfig{
		ProblemMapper: httpx.ProblemMapperFunc(mapProfileError),
		OpenAPI:       &openapi.Operation{Summary: "Get a profile"},
	})

	putHandler := httpx.Endpoint(profileStore.put, httpx.EndpointConfig{
		ProblemMapper: httpx.ProblemMapperFunc(mapProfileError),
		OpenAPI:       &openapi.Operation{Summary: "Replace a profile"},
	})

	// getHandler is wrapped in ETag *before* being used anywhere, so
	// every caller - a direct client GET, and patch.From's own internal
	// GET below - sees the same ETag. This does mean the route no
	// longer implements httpx.OpenAPIProvider (the wrapped handler is a
	// plain http.HandlerFunc), so it won't show up in /openapi.json -
	// an accepted trade-off for this example, per httpx/patch.From's
	// own doc comment ("wrap get/put themselves" is the documented way
	// to combine From with middleware).
	getHandlerWithETag := middleware.ETag()(getHandler)

	router.GET("/profiles/{id}", getHandlerWithETag)
	router.PUT("/profiles/{id}", putHandler)

	// PATCH is derived from the GET/PUT handlers above - neither needed
	// any change to support it. Using getHandlerWithETag (not the bare
	// getHandler) here is what lets patch.From's internal GET see a
	// real ETag to check the incoming PATCH's own If-Match against.
	router.PATCH("/profiles/{id}", patch.From(getHandlerWithETag, putHandler, patch.Config{}))

	document := generator.Generate()

	router.GET("/openapi.json", openapi.NewHandler(&document))
	router.GET("/docs", openapi.NewDocsHandler(openapi.DocsConfig{}))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	err := server.ListenAndServe()
	if err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
