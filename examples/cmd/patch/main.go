// Command patch demonstrates deriving PATCH from an existing GET+PUT
// pair via httpx/patch.From: RFC 7386 (JSON Merge Patch) and RFC 6902
// (JSON Patch) are both supported, selected by the incoming request's
// Content-Type, with no change to the GET/PUT handlers themselves.
//
// See httpx/patch's package doc comment for the mechanism (internal
// GET, apply the patch to the raw JSON, internal PUT), and its own
// doc comment on From for the accepted lost-update race under
// concurrent PATCHes to the same resource (arnon has no
// If-Match/optimistic-concurrency support yet).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Casara/arnon/examples/internal/logging"
	"github.com/Casara/arnon/httpx"
	"github.com/Casara/arnon/httpx/patch"
	"github.com/Casara/arnon/httpx/routing"
	"github.com/Casara/arnon/openapi"
	"github.com/Casara/arnon/problem"
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
// replacement, same as any other PUT. Mixes single-purpose tags on
// purpose (ID's path tag alone, the rest json/validate alone), same
// as examples/internal/users.GetUserRequest.
//
//nolint:tagalign // see doc comment above
type putProfileRequest struct {
	ID   string   `path:"id"`
	Name string   `          json:"name" validate:"required"`
	Bio  string   `          json:"bio"`
	Tags []string `          json:"tags"`
}

func (s *store) put(
	_ context.Context,
	request putProfileRequest,
) (Profile, error) {
	profile := Profile{
		Name: request.Name,
		Bio:  request.Bio,
		Tags: request.Tags,
	}

	s.mu.Lock()
	s.profiles[request.ID] = profile
	s.mu.Unlock()

	return profile, nil
}

func mapProfileError(err error) *problem.Problem {
	var notFound notFoundError
	if errors.As(err, &notFound) {
		return problem.NewNotFound(notFound.Error())
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
		routing.WithOpenAPI(openapi.NewRegistry(generator)),
	)

	getHandler := httpx.Endpoint(profileStore.get, httpx.EndpointConfig{
		ProblemMapper: httpx.ProblemMapperFunc(mapProfileError),
		OpenAPI:       &openapi.Operation{Summary: "Get a profile"},
	})

	putHandler := httpx.Endpoint(profileStore.put, httpx.EndpointConfig{
		ProblemMapper: httpx.ProblemMapperFunc(mapProfileError),
		OpenAPI:       &openapi.Operation{Summary: "Replace a profile"},
	})

	router.GET("/profiles/{id}", getHandler)
	router.PUT("/profiles/{id}", putHandler)

	// PATCH is derived entirely from the GET/PUT handlers above -
	// neither needed any change to support it.
	router.PATCH("/profiles/{id}", patch.From(getHandler, putHandler, patch.Config{}))

	document := generator.Generate()

	router.GET("/openapi.json", openapi.NewHandler(&document))
	router.GET("/docs", openapi.NewDocsHandler(nil))

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
