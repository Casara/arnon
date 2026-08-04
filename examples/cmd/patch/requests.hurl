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

# --- RFC 9110 write preconditions (httpx/precondition) --------------
#
# This section ends with the resource back in its original state, so
# it doesn't disturb the PATCH sections after it.

# Capture the ETag a real client would see - middleware.ETag() sets it
# on the GET route.
GET http://localhost:8080/profiles/ada
HTTP 200
[Captures]
stale_etag: header "ETag"

# A PUT with the correct If-Match succeeds and changes the resource -
# stale_etag no longer matches after this.
PUT http://localhost:8080/profiles/ada
If-Match: {{stale_etag}}
{
    "name": "Ada Lovelace",
    "bio": "Temporary bio for the If-Match demo",
    "tags": ["mathematics"]
}
HTTP 200

# A second PUT reusing the now-stale ETag is rejected (store.put's own
# precondition.Check call), and nothing is written.
PUT http://localhost:8080/profiles/ada
If-Match: {{stale_etag}}
{
    "name": "Someone Else",
    "bio": "Should never apply",
    "tags": []
}
HTTP 412
[Asserts]
header "Content-Type" == "application/problem+json; charset=utf-8"

# A PATCH with the same stale If-Match is rejected the same way - by
# patch.From itself, before the internal PUT is ever attempted.
PATCH http://localhost:8080/profiles/ada
If-Match: {{stale_etag}}
{
    "bio": "Should never apply either"
}
HTTP 412

# Confirms both rejected writes above genuinely left the resource
# untouched (still the "Temporary bio" state from the first PUT
# above).
GET http://localhost:8080/profiles/ada
HTTP 200
[Asserts]
jsonpath "$.name" == "Ada Lovelace"
jsonpath "$.bio" == "Temporary bio for the If-Match demo"

# A write with no If-Match at all still succeeds - Config.Require
# defaults to false, so a precondition is optional, not mandatory.
# Restores the profile to its original state for the sections below.
PUT http://localhost:8080/profiles/ada
{
    "name": "Ada Lovelace",
    "bio": "Mathematician and writer",
    "tags": ["mathematics"]
}
HTTP 200

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
