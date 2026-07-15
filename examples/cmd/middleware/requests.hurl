# Requests for examples/cmd/middleware.
#
# Run with the example server up (`go run ./examples/cmd/middleware`):
#
#   hurl --test examples/cmd/middleware/requests.hurl

# --- POST /api/users --------------------------------------------------

# Success. Also proves several global/group middlewares are wired up:
# SecureHeaders (response headers), RateLimit (X-RateLimit-* headers),
# NoCache (Cache-Control) - all without needing to trip any of them.
POST http://localhost:8080/api/users
Content-Type: application/json
{
    "name": "Ada Lovelace",
    "email": "ada@example.com"
}
HTTP 201
[Asserts]
header "Content-Type" == "application/json; charset=utf-8"
jsonpath "$.id" exists
header "X-Content-Type-Options" == "nosniff"
header "X-Frame-Options" == "DENY"
header "Cache-Control" contains "no-store"
header "X-RateLimit-Limit" == "100"
header "X-RateLimit-Remaining" exists

# Trailing slash: StripSlashes normalizes the path before routing, so
# this hits the same handler as above instead of 404ing.
POST http://localhost:8080/api/users/
Content-Type: application/json
{
    "name": "Ada Lovelace",
    "email": "ada@example.com"
}
HTTP 201
[Asserts]
jsonpath "$.id" exists

# Missing name: binds fine (zero value), fails `required`.
POST http://localhost:8080/api/users
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
POST http://localhost:8080/api/users
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
POST http://localhost:8080/api/users
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
POST http://localhost:8080/api/users
Content-Type: application/json
```
{"name":
```
HTTP 400
[Asserts]
jsonpath "$.title" == "Request validation failed"

# Unsupported Content-Type: AllowContentType only accepts
# "application/json" on the /api group.
POST http://localhost:8080/api/users
Content-Type: text/plain
```
Ada Lovelace
```
HTTP 415
[Asserts]
jsonpath "$.title" == "Unsupported Media Type"

# Oversized body: MaxBodyBytes is configured at 1024 bytes for this
# example, and Content-Length is known up front here, so the request
# is rejected immediately, before binding ever runs.
POST http://localhost:8080/api/users
Content-Type: application/json
{
    "name": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "email": "ada@example.com"
}
HTTP 413
[Asserts]
jsonpath "$.title" == "Payload Too Large"

# Compressed response: Compress only kicks in when the client accepts
# gzip and the response Content-Type is compressible (application/json
# is, by default).
POST http://localhost:8080/api/users
Content-Type: application/json
Accept-Encoding: gzip
{
    "name": "Ada Lovelace",
    "email": "ada@example.com"
}
HTTP 201
[Asserts]
header "Content-Encoding" == "gzip"

# --- GET /openapi.json ------------------------------------------------

GET http://localhost:8080/openapi.json
HTTP 200
[Asserts]
jsonpath "$.openapi" == "3.2.0"
jsonpath "$.paths['/api/users'].post" exists

# --- GET /docs ----------------------------------------------------------

GET http://localhost:8080/docs
HTTP 200
[Asserts]
header "Content-Type" == "text/html; charset=utf-8"
