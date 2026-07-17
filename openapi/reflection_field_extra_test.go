package openapi_test

import (
	"testing"

	"github.com/Casara/arnon/openapi"
)

// TestParseTypedValue_ConvertsEveryKindItSupports drives
// parseTypedValue (via the `example`/`default` struct tags) through
// each reflect.Kind branch it recognizes: bool, every signed/unsigned
// integer width it groups together, and float. String and int are
// already covered by reflection_test.go.
func TestParseTypedValue_ConvertsEveryKindItSupports(t *testing.T) {
	t.Parallel()

	type request struct {
		Active   bool    `default:"false" example:"true" json:"active"`
		Count    uint    `default:"0"     example:"7"    json:"count"`
		Ratio    float64 `default:"0.5"   example:"1.5"  json:"ratio"`
		Nickname *string `                example:"nick" json:"nickname"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	active := schema.Properties["active"]

	if active.Example != true {
		t.Errorf("expected bool example true, got %#v", active.Example)
	}

	if active.Default != false {
		t.Errorf("expected bool default false, got %#v", active.Default)
	}

	count := schema.Properties["count"]

	if count.Example != uint64(7) {
		t.Errorf("expected uint example uint64(7), got %#v", count.Example)
	}

	ratio := schema.Properties["ratio"]

	if ratio.Example != 1.5 {
		t.Errorf("expected float example 1.5, got %#v", ratio.Example)
	}

	// Pointer fields must be unwrapped before parseTypedValue inspects
	// the underlying kind (here, string).
	nickname := schema.Properties["nickname"]

	if nickname.Example != "nick" {
		t.Errorf("expected pointer-field example %q, got %#v", "nick", nickname.Example)
	}
}

// TestParseTypedValue_UnparsableValueFallsBackToRawString covers the
// case where the tag value doesn't parse as the field's Go kind (e.g.
// "not-a-bool" for a bool field): parseTypedValue falls through to
// returning the raw string instead of the typed zero value.
func TestParseTypedValue_UnparsableValueFallsBackToRawString(t *testing.T) {
	t.Parallel()

	type request struct {
		Active bool `example:"not-a-bool"   json:"active"`
		Count  int  `example:"not-a-number" json:"count"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	if got := schema.Properties["active"].Example; got != "not-a-bool" {
		t.Errorf("expected raw string fallback %q, got %#v", "not-a-bool", got)
	}

	if got := schema.Properties["count"].Example; got != "not-a-number" {
		t.Errorf("expected raw string fallback %q, got %#v", "not-a-number", got)
	}
}

// TestParseTypedValue_UnsupportedKindReturnsRawString confirms a field
// type outside the switch (here, a struct) also falls back to the raw
// tag string.
func TestParseTypedValue_UnsupportedKindReturnsRawString(t *testing.T) {
	t.Parallel()

	type nested struct {
		X int
	}

	type request struct {
		Nested nested `example:"{}" json:"nested"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	if got := schema.Properties["nested"].Example; got != "{}" {
		t.Errorf("expected raw string fallback %q, got %#v", "{}", got)
	}
}

// TestInferFormatFromValidator_RecognizesEveryFormat covers every
// remaining branch of inferFormatFromValidator beyond uuid/uuid4/
// datetime (already exercised elsewhere): hostname, ipv4, ipv6, ip,
// date, and a tag with no known format at all (returns ""). One field
// per case, all read off a single generated schema, since struct tags
// can't be built dynamically in Go.
//
// "uri" (rather than "url") is used for the URI case on purpose:
// applyValidationTags only special-cases the literal tag "url" (via
// applyEmail/applyUUID/applyURL, which run first and set schema.Format
// directly); "uri" isn't one of those, so it reaches
// inferFormatFromValidator's own "url", "uri" case instead. The
// literal tags "email" and "url" would never reach
// inferFormatFromValidator at all - applyValidationTags always sets
// schema.Format for them first, and parseField only calls
// inferFormatFromValidator when schema.Format is still empty. That
// makes inferFormatFromValidator's own "email" case dead code today;
// see the final report.
func TestInferFormatFromValidator_RecognizesEveryFormat(t *testing.T) {
	t.Parallel()

	type request struct {
		UUID4        string `json:"uuid4"        validate:"uuid4"`
		URI          string `json:"uri"          validate:"uri"`
		Hostname     string `json:"hostname"     validate:"hostname"`
		IPv4         string `json:"ipv4"         validate:"ipv4"`
		IPv6         string `json:"ipv6"         validate:"ipv6"`
		IP           string `json:"ip"           validate:"ip"`
		Date         string `json:"date"         validate:"date"`
		Unrecognized string `json:"unrecognized" validate:"not_a_format_tag"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	wantFormats := map[string]string{
		"uuid4":        "uuid",
		"uri":          "uri",
		"hostname":     "hostname",
		"ipv4":         "ipv4",
		"ipv6":         "ipv6",
		"ip":           "ip",
		"date":         "date",
		"unrecognized": "",
	}

	for field, wantFormat := range wantFormats {
		if got := schema.Properties[field].Format; got != wantFormat {
			t.Errorf("expected field %q format %q, got %q", field, wantFormat, got)
		}
	}
}
