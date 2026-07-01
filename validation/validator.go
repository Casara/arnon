package validation

import "github.com/Casara/arnon/problem"

// Validator validates a value and returns validation problems.
type Validator interface {
	Validate(value any) []problem.ValidationError
}
