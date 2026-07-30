// Package customvalidators registers the custom validation rules
// shared by every runnable example under examples/cmd.
package customvalidators

import (
	"strings"

	"github.com/casara/arnon/validation"
)

// RegisterCustomValidators registers the framework-wide custom
// validation rules used by the examples.
//
// It must run before the first validator is built (i.e. before the
// first httpx.Endpoint call), since validation.Default() is a
// lazily-initialized singleton that only picks up custom rules
// registered before its first use.
func RegisterCustomValidators() {
	err := validation.RegisterCustomRule(validation.CustomRule{
		// validator/v10 has no built-in tag for "not just whitespace":
		// required only rejects the Go zero value (""), so a name of
		// " " passes it. notblank fills that specific gap.
		Tag: "notblank",

		Func: func(field validation.FieldContext) bool {
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
