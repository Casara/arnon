# Requests for examples/cmd/patch.
#
# Run with the example server up (`go run ./examples/cmd/patch`):
#
#   hurl --test examples/cmd/patch/requests.hurl
#
# Requests run in order and share state (an in-memory store), so
# later ones depend on earlier ones having applied.

# --- GET /profiles/ada (initial state) ----------------------------

GET http://localhost:8080/profiles/ada
HTTP 200
[Asserts]
jsonpath "$.name" == "Ada Lovelace"
jsonpath "$.bio" == "Mathematician and writer"
jsonpath "$.tags" count == 1
jsonpath "$.tags[0]" == "mathematics"

# --- PATCH: RFC 7386 JSON Merge Patch -----------------------------

# No Content-Type: merge patch is the default.
PATCH http://localhost:8080/profiles/ada
{
    "bio": "Countess of Lovelace"
}
HTTP 200
[Asserts]
jsonpath "$.bio" == "Countess of Lovelace"
jsonpath "$.name" == "Ada Lovelace"

# Explicit Content-Type, and "null" deletes the field (the distinction
# a plain Go struct field can't express - see httpx/patch's doc
# comment).
PATCH http://localhost:8080/profiles/ada
Content-Type: application/merge-patch+json
{
    "bio": null
}
HTTP 200
[Asserts]
jsonpath "$.bio" not exists
jsonpath "$.name" == "Ada Lovelace"

# --- PATCH: RFC 6902 JSON Patch -------------------------------------

PATCH http://localhost:8080/profiles/ada
Content-Type: application/json-patch+json
[
    {"op": "add", "path": "/tags/-", "value": "computing"},
    {"op": "replace", "path": "/name", "value": "Ada, Countess of Lovelace"}
]
HTTP 200
[Asserts]
jsonpath "$.name" == "Ada, Countess of Lovelace"
jsonpath "$.tags" count == 2
jsonpath "$.tags[1]" == "computing"

# Confirms every patch above actually persisted (each PATCH replayed
# a real PUT, not just an in-memory response).
GET http://localhost:8080/profiles/ada
HTTP 200
[Asserts]
jsonpath "$.name" == "Ada, Countess of Lovelace"
jsonpath "$.bio" not exists
jsonpath "$.tags" count == 2
jsonpath "$.tags[1]" == "computing"

# --- Error cases ------------------------------------------------------

# The internal GET reports the resource as missing - that response is
# forwarded as-is, the internal PUT is never attempted.
PATCH http://localhost:8080/profiles/turing
{
    "bio": "x"
}
HTTP 404

# Malformed patch body.
PATCH http://localhost:8080/profiles/ada
Content-Type: application/json
```
{not valid json}
```
HTTP 400
[Asserts]
header "Content-Type" == "application/problem+json; charset=utf-8"

# Unsupported Content-Type.
PATCH http://localhost:8080/profiles/ada
Content-Type: application/xml
<name>Ada</name>
HTTP 415

# A syntactically valid merge patch that produces an invalid document
# (Name is validate:"required") still goes through the real PUT
# handler's own validation, unchanged - the same RFC 9457 + RFC 6901
# error shape as a direct PUT would produce.
PATCH http://localhost:8080/profiles/ada
{
    "name": ""
}
HTTP 400
[Asserts]
jsonpath "$.errors[0].code" == "required"
jsonpath "$.errors[0].source.field" == "/name"
