package openapi_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/casara/arnon/openapi"
)

// A 3.1+ document has no "nullable" keyword: the specification removed it, and
// a conforming parser ignores it, so emitting it would silently lose the
// information in a document that declares 3.2.0.
func TestSchema_MarshalJSON_NullableBecomesTypeArray(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(openapi.Schema{Type: "string", Nullable: true})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if string(encoded) != `{"type":["string","null"]}` {
		t.Errorf("got %s", encoded)
	}

	if bytes.Contains(encoded, []byte("nullable")) {
		t.Errorf("3.0 nullable keyword leaked: %s", encoded)
	}
}

// A bare $ref has no type to widen, so Nullable is dropped rather than
// emitting a meaningless ["null"].
func TestSchema_MarshalJSON_NullableWithoutTypeIsDropped(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(openapi.Schema{
		Ref:      "#/components/schemas/User",
		Nullable: true,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if string(encoded) != `{"$ref":"#/components/schemas/User"}` {
		t.Errorf("got %s", encoded)
	}
}
