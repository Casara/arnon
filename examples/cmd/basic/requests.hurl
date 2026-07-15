# Requests for examples/cmd/basic.
#
# Run with the example server up (`go run ./examples/cmd/basic`):
#
#   hurl --test examples/cmd/basic/requests.hurl

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

# Missing name: binds fine (zero value), fails `required`.
POST http://localhost:8080/users
Content-Type: application/json
{
    "email": "ada@example.com"
}
HTTP 400
[Asserts]
header "Content-Type" == "application/problem+json; charset=utf-8"
jsonpath "$.errors[0].code" == "required"
jsonpath "$.errors[0].source.field" == "/name"

# Whitespace-only name: passes `required` (non-zero value), fails the
# custom `notblank` rule instead.
POST http://localhost:8080/users
Content-Type: application/json
{
    "name": "   ",
    "email": "ada@example.com"
}
HTTP 400
[Asserts]
jsonpath "$.errors[0].code" == "blank"
jsonpath "$.errors[0].source.field" == "/name"

# Invalid email.
POST http://localhost:8080/users
Content-Type: application/json
{
    "name": "Ada Lovelace",
    "email": "not-an-email"
}
HTTP 400
[Asserts]
jsonpath "$.errors[0].code" == "invalid_email"
jsonpath "$.errors[0].source.field" == "/email"

# Malformed JSON body.
POST http://localhost:8080/users
Content-Type: application/json
```
{"name":
```
HTTP 400
[Asserts]
jsonpath "$.title" == "Request validation failed"

# --- GET /openapi.json ------------------------------------------------

GET http://localhost:8080/openapi.json
HTTP 200
[Asserts]
jsonpath "$.openapi" == "3.2.0"
jsonpath "$.paths['/users'].post" exists

# --- GET /docs ----------------------------------------------------------

GET http://localhost:8080/docs
HTTP 200
[Asserts]
header "Content-Type" == "text/html; charset=utf-8"
