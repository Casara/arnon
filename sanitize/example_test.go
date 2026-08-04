package sanitize_test

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/casara/arnon/sanitize"
)

// The sanitize tag chains named transforms, applied left to right. Two are
// built in: "trim" and "email" (trim plus lowercase). A nested struct is always
// recursed into; a slice, array or map needs its tag to start with "dive",
// matching validator/v10's own convention.
func ExampleApply() {
	type address struct {
		City string `sanitize:"trim"`
	}

	type signup struct {
		Email   string   `sanitize:"email"`
		Name    string   `sanitize:"trim"`
		Tags    []string `sanitize:"dive,trim"`
		Address address
	}

	form := signup{
		Email:   "  ADA@Example.COM ",
		Name:    "  Ada Lovelace  ",
		Tags:    []string{" Go ", "  HTTP"},
		Address: address{City: "  London  "},
	}

	sanitize.Apply(&form)

	fmt.Printf("%q\n", form.Email)
	fmt.Printf("%q\n", form.Name)
	fmt.Printf("%q\n", form.Tags)
	fmt.Printf("%q\n", form.Address.City)
	// Output:
	// "ada@example.com"
	// "Ada Lovelace"
	// ["Go" "HTTP"]
	// "London"
}

// Prepare validates every sanitize tag on a type without needing a value, so a
// typo fails at startup instead of silently doing nothing on the one request
// that happens to populate the field. httpx.Endpoint calls it for you.
func ExamplePrepare() {
	type good struct {
		Email string `sanitize:"email"`
	}

	type typo struct {
		Email string `sanitize:"trimm"`
	}

	fmt.Println(sanitize.Prepare(reflect.TypeFor[good]()))
	fmt.Println(sanitize.Prepare(reflect.TypeFor[typo]()))
	// Output:
	// <nil>
	// Email: sanitizer "trimm": sanitize: unknown sanitizer
}

// RegisterFunc adds a transform under a new tag name, usable from any struct
// from then on. Call it during bootstrap, before the first Endpoint that uses
// the tag is constructed. FromRegexp builds a "delete everything that matches"
// transform.
func ExampleRegisterFunc() {
	err := sanitize.RegisterFunc("nodigits", sanitize.FromRegexp(
		regexp.MustCompile(`\d`),
	))
	if err != nil {
		fmt.Println("register:", err)

		return
	}

	err = sanitize.RegisterFunc("dashes", func(value string) string {
		return strings.ReplaceAll(value, " ", "-")
	})
	if err != nil {
		fmt.Println("register:", err)

		return
	}

	type product struct {
		Slug string `sanitize:"trim,nodigits,dashes"`
	}

	item := product{Slug: "  Widget 2000 Pro  "}

	sanitize.Apply(&item)

	fmt.Printf("%q\n", item.Slug)
	// Output:
	// "Widget--Pro"
}
