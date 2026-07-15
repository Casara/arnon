# Requests for examples/cmd/observability.
#
# The example itself works without a running collector (OTLP export
# failures are async and never block the HTTP response), so this
# suite only exercises the HTTP surface, not the telemetry pipeline.
#
# Run with the example server up (`go run ./examples/cmd/observability`):
#
#   hurl --test examples/cmd/observability/requests.hurl

# --- POST /api/users --------------------------------------------------

# Success.
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
