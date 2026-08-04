# Requests for examples/cmd/custom-errors.
#
# Run with the example server up (`go run ./examples/cmd/custom-errors`):
#
#   hurl --test examples/cmd/custom-errors/requests.hurl

# --- POST /users --------------------------------------------------

# Success: same handler and response body as examples/cmd/basic, since
# CreateUser/CreateUserResponse are untouched - only the error path differs.
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

# Missing name: a flat {"errors": [...]} body, not RFC 9457 - no `type`,
# `title`, `status` or `instance`. Content-Type is plain application/json,
# not application/problem+json.
POST http://localhost:8080/users
Content-Type: application/json
{
    "email": "ada@example.com"
}
HTTP 400
[Asserts]
header "Content-Type" == "application/json; charset=utf-8"
jsonpath "$.title" not exists
jsonpath "$.errors[0].field" == "/name"
jsonpath "$.errors[0].code" == "required"

# Whitespace-only name: fails the customvalidators.RegisterCustomValidators
# "notblank" rule, same as examples/cmd/basic - registered custom rules work
# unmodified through this hand-rolled pipeline, since it calls the same
# validation.Default() singleton httpx.Endpoint would.
POST http://localhost:8080/users
Content-Type: application/json
{
    "name": "   ",
    "email": "ada@example.com"
}
HTTP 400
[Asserts]
jsonpath "$.errors[0].field" == "/name"
jsonpath "$.errors[0].code" == "blank"

# Malformed JSON body: a binding failure, handled the same way as a
# validation failure - both come out of binding.Decode/validation.Validate
# as []problem.ValidationError before writeErrors ever sees them.
POST http://localhost:8080/users
Content-Type: application/json
```
{"name":
```
HTTP 400
[Asserts]
jsonpath "$.errors[0].field" == "/"
