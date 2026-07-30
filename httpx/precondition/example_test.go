package precondition_test

import (
	"fmt"
	"time"

	"github.com/casara/arnon/httpx/precondition"
)

// Check compares the client's If-Match against the resource's *current* ETag,
// which the caller supplies: only the code that just loaded the row knows it.
// A nil result means the write may proceed.
func ExampleCheck() {
	const currentETag = `"v2"`

	stale := precondition.Check(
		`"v1"`,
		"",
		currentETag,
		time.Time{},
		precondition.Config{},
	)
	fmt.Println(stale.StatusCode(), stale.Detail)

	fresh := precondition.Check(
		currentETag,
		"",
		currentETag,
		time.Time{},
		precondition.Config{},
	)
	fmt.Println(fresh)
	// Output:
	// 412 the resource's current state does not match If-Match
	// <nil>
}

// If-Match uses strong comparison (RFC 9110 §13.1.1), so a weak validator can
// never satisfy it - unlike If-None-Match on a GET, which compares weakly.
func ExampleCheck_weakETag() {
	weak := precondition.Check(
		`W/"v2"`,
		"",
		`"v2"`,
		time.Time{},
		precondition.Config{},
	)

	fmt.Println(weak.StatusCode())
	// Output:
	// 412
}

// By default a request carrying neither precondition header proceeds.
// Config.Require turns that into 428 Precondition Required, for an API where
// every write must be conditional.
func ExampleConfig() {
	permissive := precondition.Check(
		"", "", `"v1"`, time.Time{},
		precondition.Config{},
	)
	fmt.Println(permissive)

	required := precondition.Check(
		"", "", `"v1"`, time.Time{},
		precondition.Config{Require: true},
	)
	fmt.Println(required.StatusCode(), required.Detail)
	// Output:
	// <nil>
	// 428 this request requires an If-Match or If-Unmodified-Since header
}
