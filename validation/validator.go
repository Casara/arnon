package validation

import "github.com/casara/arnon/problem"

// Validator validates a value and returns validation problems.
type Validator interface {
	Validate(value any) []problem.ValidationError
}
