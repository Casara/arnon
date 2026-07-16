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
# Problem.Instance is auto-populated from the request path.
jsonpath "$.instance" == "/users"

# Accept negotiation: an Accept header that explicitly excludes
# application/json gets a 406, since this endpoint only produces JSON.
POST http://localhost:8080/users
Content-Type: application/json
Accept: application/xml
{
    "name": "Ada Lovelace",
    "email": "ada@example.com"
}
HTTP 406
[Asserts]
header "Content-Type" == "application/problem+json; charset=utf-8"

# --- GET /users/{id} ----------------------------------------------------

# Path binding plus []string binding from query (repeated keys) and
# header (Accept-Language, comma-joined in a single line here).
GET http://localhost:8080/users/42?tags=admin&tags=beta
Accept-Language: en, fr
HTTP 200
[Asserts]
jsonpath "$.id" == "42"
jsonpath "$.tags" count == 2
jsonpath "$.tags[0]" == "admin"
jsonpath "$.tags[1]" == "beta"
jsonpath "$.locales" count == 2
jsonpath "$.locales[0]" == "en"
jsonpath "$.locales[1]" == "fr"

# Same endpoint, no query/header values: both []string fields come
# back empty rather than causing a binding error.
GET http://localhost:8080/users/42
HTTP 200
[Asserts]
jsonpath "$.id" == "42"

# --- GET /openapi.json ------------------------------------------------

GET http://localhost:8080/openapi.json
HTTP 200
[Asserts]
jsonpath "$.openapi" == "3.2.0"
jsonpath "$.paths['/users'].post" exists
jsonpath "$.paths['/users/{id}'].get" exists

# --- GET /docs ----------------------------------------------------------

GET http://localhost:8080/docs
HTTP 200
[Asserts]
header "Content-Type" == "text/html; charset=utf-8"
