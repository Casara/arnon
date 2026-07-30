package patch_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/casara/arnon/httpx/patch"
	"github.com/casara/arnon/httpx/routing"
)

type profile struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	City  string `json:"city"`
}

// stored stands in for whatever storage the real GET and PUT handlers use.
//
//nolint:gochecknoglobals // example fixture, stands in for a real datastore
var stored = profile{
	Name:  "Ada Lovelace",
	Email: "ada@example.com",
	City:  "London",
}

func getProfile(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(writer).Encode(stored)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}

func putProfile(writer http.ResponseWriter, request *http.Request) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)

		return
	}

	var replacement profile

	err = json.Unmarshal(body, &replacement)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)

		return
	}

	stored = replacement

	writer.WriteHeader(http.StatusNoContent)
}

// From derives a PATCH handler from an existing GET and PUT pair, by replaying
// the request: it fetches the current representation with get, applies the
// patch document to those bytes, and replays the result through put. Neither
// handler needs to know anything about patching. RFC 7386 (JSON Merge Patch)
// and RFC 6902 (JSON Patch) are selected by Content-Type.
func ExampleFrom() {
	router := routing.NewRouter()
	router.GET("/profile", http.HandlerFunc(getProfile))
	router.PUT("/profile", http.HandlerFunc(putProfile))
	router.PATCH("/profile", patch.From(
		http.HandlerFunc(getProfile),
		http.HandlerFunc(putProfile),
		patch.Config{},
	))

	request := httptest.NewRequest(
		http.MethodPatch,
		"/profile",
		strings.NewReader(`{"city":"Paris"}`),
	)
	request.Header.Set("Content-Type", "application/merge-patch+json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	fmt.Println("status:", recorder.Code)
	fmt.Printf("%+v\n", stored)
	// Output:
	// status: 204
	// {Name:Ada Lovelace Email:ada@example.com City:Paris}
}

// An unsupported Content-Type is rejected before either handler runs.
func ExampleFrom_unsupportedContentType() {
	handler := patch.From(
		http.HandlerFunc(getProfile),
		http.HandlerFunc(putProfile),
		patch.Config{},
	)

	request := httptest.NewRequest(
		http.MethodPatch,
		"/profile",
		strings.NewReader(`{"city":"Paris"}`),
	)
	request.Header.Set("Content-Type", "text/plain")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	fmt.Println(recorder.Code)
	// Output:
	// 415
}
