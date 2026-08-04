# Requests for examples/cmd/custom-validator.
#
# Run with the example server up (`go run ./examples/cmd/custom-validator`):
#
#   hurl --test examples/cmd/custom-validator/requests.hurl

# --- POST /users --------------------------------------------------

# Success.
POST http://localhost:8080/users
Content-Type: application/json
{
    "name": "Ada Lovelace",
    "email": "ada@example.com"
}
HTTP 201
[Asserts]
header "Content-Type" == "application/json; charset=utf-8"
jsonpath "$.id" exists

# Missing name: ozzo-validation's Required rule fails it, with ozzo's own
# code ("validation_required") - not arnon's own "required" - since the
# code comes from whatever validator is plugged in, not from arnon itself.
POST http://localhost:8080/users
Content-Type: application/json
{
    "email": "ada@example.com"
}
HTTP 400
[Asserts]
header "Content-Type" == "application/problem+json; charset=utf-8"
jsonpath "$.errors[0].code" == "validation_required"
jsonpath "$.errors[0].source.field" == "/name"

# Invalid email: ozzo-validation's is.Email rule, same swapped-code point as
# above ("validation_is_email").
POST http://localhost:8080/users
Content-Type: application/json
{
    "name": "Ada Lovelace",
    "email": "not-an-email"
}
HTTP 400
[Asserts]
jsonpath "$.errors[0].code" == "validation_is_email"
jsonpath "$.errors[0].source.field" == "/email"

# Email with surrounding whitespace and mixed case: sanitize:"email" still
# runs before Validate(), since sanitize is independent of which Validator
# is plugged in - swapping the validator does not opt out of sanitization.
POST http://localhost:8080/users
Content-Type: application/json
{
    "name": "Ada Lovelace",
    "email": "  Ada@Example.com  "
}
HTTP 201
[Asserts]
jsonpath "$.id" exists

# --- GET /openapi.json ------------------------------------------------

# The generated schema has neither `required` nor `format: email` on these
# fields: OpenAPI generation reads the `validate` struct tag directly
# (openapi/reflection_field.go), independently of which Validator runs at
# request time. CreateUserRequest here carries no `validate` tag - the rules
# live in Validate() instead - so nothing feeds the schema, even though the
# rules are enforced at request time (see the two 400 cases above). This is
# the concrete cost of swapping the validator: the generated docs stop
# reflecting the active rules unless kept in sync by hand.
GET http://localhost:8080/openapi.json
HTTP 200
[Asserts]
jsonpath "$.components.schemas.CreateUserRequest.required" not exists
jsonpath "$.components.schemas.CreateUserRequest.properties.email.format" not exists
