package validation_test

import (
	"fmt"
	"strings"

	validatorv10 "github.com/go-playground/validator/v10"

	"github.com/casara/arnon/validation"
)

// The default validator reads validate struct tags and reports every failure at
// once - it never stops at the first one - with each error pointing at the
// field through an RFC 6901 JSON Pointer built from the json tag.
func ExampleDefault() {
	type signup struct {
		Name  string `json:"name"  validate:"required"`
		Email string `json:"email" validate:"required,email"`
		Age   int    `json:"age"   validate:"gte=18"`
	}

	failures := validation.Default().Validate(signup{Email: "nope", Age: 12})

	fmt.Println("failures:", len(failures))

	for _, failure := range failures {
		fmt.Printf("%s %s\n", failure.Source.Field, failure.Code)
	}
	// Output:
	// failures: 3
	// /name required
	// /email invalid_email
	// /age greater_than_or_equal
}

// RegisterCustomRule is the single place a custom tag is defined. One
// registration feeds all three surfaces that must agree on what the tag means:
// runtime validation (Func), the problem detail (Code and Message) and the
// generated OpenAPI schema (Schema).
func ExampleRegisterCustomRule() {
	err := validation.RegisterCustomRule(validation.CustomRule{
		Tag: "notblank",
		Func: func(field validatorv10.FieldLevel) bool {
			return strings.TrimSpace(field.Field().String()) != ""
		},
		Code:    "blank",
		Message: func(string) string { return "must not be blank" },
		Schema:  &validation.SchemaEffect{Pattern: `\S`},
	})
	if err != nil {
		fmt.Println("register:", err)

		return
	}

	type comment struct {
		Body string `json:"body" validate:"notblank"`
	}

	validator, err := validation.New()
	if err != nil {
		fmt.Println("new:", err)

		return
	}

	failures := validator.Validate(comment{Body: "   "})

	for _, failure := range failures {
		fmt.Printf("%s %s %q\n", failure.Source.Field, failure.Code, failure.Detail)
	}

	rule, found := validation.LookupCustomRule("notblank")

	fmt.Println(found, rule.Schema.Pattern)
	// Output:
	// /body blank "must not be blank"
	// true \S
}
