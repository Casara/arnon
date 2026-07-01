package problem

// ValidationLocation identifies where the validation error happened.
type ValidationLocation string

// Validation locations identify where a validation error originated.
const (
	ValidationLocationBody ValidationLocation = "body"

	ValidationLocationPath ValidationLocation = "path"

	ValidationLocationQuery ValidationLocation = "query"

	ValidationLocationHeader ValidationLocation = "header"
)

// ValidationSource describes the source of a validation error.
type ValidationSource struct {
	In ValidationLocation `json:"in,omitempty"`

	Field string `json:"field,omitempty"`
}
