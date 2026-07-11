package main

import (
	"strings"

	validatorv10 "github.com/go-playground/validator/v10"

	"github.com/Casara/arnon/validation"
)

// registerCustomValidators registers the framework-wide custom
// validation rules used by this example.
//
// It must run before the first validator is built (i.e. before the
// first httpx.Endpoint call), since validation.Default() is a
// lazily-initialized singleton that only picks up custom rules
// registered before its first use.
func registerCustomValidators() {
	err := validation.RegisterCustomRule(validation.CustomRule{
		// validator/v10 has no built-in tag for "not just whitespace":
		// required only rejects the Go zero value (""), so a name of
		// " " passes it. notblank fills that specific gap.
		Tag: "notblank",

		Func: func(field validatorv10.FieldLevel) bool {
			return strings.TrimSpace(field.Field().String()) != ""
		},

		Schema: &validation.SchemaEffect{
			Pattern: `.*\S.*`,
		},

		Code: "blank",

		Message: func(string) string {
			return "must not be blank"
		},
	})
	if err != nil {
		panic(err)
	}
}
