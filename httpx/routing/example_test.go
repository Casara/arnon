package routing_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/casara/arnon/httpx/routing"
)

func write(text string) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(writer, text)
	}
}

// A Router is an http.Handler, so it goes straight into an http.Server with no
// adapter. Path parameters use net/http's own {name} syntax and are read with
// request.PathValue.
func ExampleNewRouter() {
	router := routing.NewRouter()

	router.GET("/users/{id}", http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		fmt.Println("id =", request.PathValue("id"))

		writer.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/users/42", nil),
	)

	fmt.Println(recorder.Code)

	// An unregistered method on a registered path is answered by net/http's own
	// ServeMux, with a real Allow header. HEAD is listed because ServeMux
	// serves it from the GET registration for free.
	recorder = httptest.NewRecorder()
	router.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodDelete, "/users/42", nil),
	)

	fmt.Println(recorder.Code, recorder.Header().Get("Allow"))
	// Output:
	// id = 42
	// 200
	// 405 GET, HEAD
}

// Group applies a shared prefix and its own middleware to the routes declared
// inside it, without affecting the rest of the router.
func ExampleRouter_Group() {
	router := routing.NewRouter()

	router.GET("/health", write("ok"))

	api := router.Group("/api/v1")
	api.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			writer.Header().Set("X-Api-Version", "v1")
			next.ServeHTTP(writer, request)
		})
	})
	api.GET("/users", write("users"))

	for _, path := range []string{"/health", "/api/v1/users"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(
			recorder,
			httptest.NewRequest(http.MethodGet, path, nil),
		)

		fmt.Printf(
			"%s -> %s (X-Api-Version: %q)\n",
			path,
			recorder.Body.String(),
			recorder.Header().Get("X-Api-Version"),
		)
	}
	// Output:
	// /health -> ok (X-Api-Version: "")
	// /api/v1/users -> users (X-Api-Version: "v1")
}

// Middleware registered with Router.Use wraps the whole mux, not each route, so
// it also runs for requests that match no route at all.
func ExampleRouter_Use() {
	router := routing.NewRouter()

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			writer.Header().Set("X-Request-Id", "req_1")
			next.ServeHTTP(writer, request)
		})
	})

	router.GET("/known", write("hit"))

	for _, path := range []string{"/known", "/unknown"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(
			recorder,
			httptest.NewRequest(http.MethodGet, path, nil),
		)

		fmt.Printf(
			"%s -> %d, X-Request-Id: %q\n",
			path,
			recorder.Code,
			recorder.Header().Get("X-Request-Id"),
		)
	}
	// Output:
	// /known -> 200, X-Request-Id: "req_1"
	// /unknown -> 404, X-Request-Id: "req_1"
}
