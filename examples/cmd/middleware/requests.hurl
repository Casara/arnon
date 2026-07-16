# Requests for examples/cmd/middleware.
#
# Run with the example server up (`go run ./examples/cmd/middleware`):
#
#   hurl --test examples/cmd/middleware/requests.hurl

# --- POST /api/users --------------------------------------------------

# Success. Also proves several global/group middlewares are wired up:
# SecureHeaders (response headers), RateLimit (X-RateLimit-* headers),
# ServiceDesc (Link), serverBrandMiddleware (X-Powered-By, a custom
# middleware wired in via ChainConfig.Extra, see main.go) - all
# without needing to trip any of them.
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
header "X-RateLimit-Limit" == "100"
header "X-RateLimit-Remaining" exists
header "Link" == "</openapi.json>; rel=\"service-desc\""
header "X-Powered-By" == "arnon-example"

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

# --- GET /api/users/{id} -------------------------------------------------

# First request: gets a full body and an ETag.
GET http://localhost:8080/api/users/42
HTTP 200
[Asserts]
jsonpath "$.id" == "42"
header "X-Powered-By" == "arnon-example"
[Captures]
userETag: header "ETag"

# Second request, with If-None-Match set to the previous ETag: the
# representation hasn't changed, so ETag (installed before Compress,
# see main.go) replaces the response with an empty-bodied 304. The
# custom serverBrandMiddleware header still shows up here too, since
# Extra anchored it before ETag (see main.go and ChainConfig.Extra).
GET http://localhost:8080/api/users/42
If-None-Match: {{userETag}}
HTTP 304
[Asserts]
header "ETag" == "{{userETag}}"
header "X-Powered-By" == "arnon-example"
body == ""

# --- OPTIONS /api/users ---------------------------------------------------

# A bare OPTIONS (no Access-Control-Request-Method, so not a CORS
# preflight) falls through CORS and reaches the mux, which answers
# with the real Allow for this path - not a fixed list from CORSConfig.
OPTIONS http://localhost:8080/api/users
HTTP 405
[Asserts]
header "Allow" contains "POST"

# --- GET /openapi.json ------------------------------------------------

GET http://localhost:8080/openapi.json
HTTP 200
[Asserts]
jsonpath "$.openapi" == "3.2.0"
jsonpath "$.paths['/api/users'].post" exists
jsonpath "$.paths['/api/users/{id}'].get" exists

# --- GET /docs ----------------------------------------------------------

GET http://localhost:8080/docs
HTTP 200
[Asserts]
header "Content-Type" == "text/html; charset=utf-8"
