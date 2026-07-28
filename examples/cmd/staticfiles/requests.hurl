# Requests for examples/cmd/staticfiles.
#
# Run with the example server up (`go run ./examples/cmd/staticfiles`):
#
#   hurl --test examples/cmd/staticfiles/requests.hurl

# --- GET /static/ -----------------------------------------------------

# http.FileServer serves assets/index.html for the directory root.
# ETag is global here (see main.go) and applies to /static/ too: a
# plain GET (no Range) gets a validator like any other route.
GET http://localhost:8080/static/
HTTP 200
[Asserts]
header "Content-Type" == "text/html; charset=utf-8"
header "Accept-Ranges" == "bytes"
header "Etag" exists

# /static/index.html itself redirects to /static/ - http.FileServer's
# own behavior for a file literally named "index.html", nothing
# arnon-specific.
GET http://localhost:8080/static/index.html
HTTP 301
[Asserts]
header "Location" == "./"

# --- GET /static/style.css ---------------------------------------------

# Plain GET with Accept-Encoding: gzip - Compress applies here too,
# same as ETag above.
GET http://localhost:8080/static/style.css
Accept-Encoding: gzip
HTTP 200
[Asserts]
header "Content-Encoding" == "gzip"
header "Etag" exists

# A Range request: ETag/Compress both step aside for it (see their
# doc comments in httpx/middleware/etag.go and compress.go), so
# http.FileServer's own Range support runs untouched - no Etag, no
# Content-Encoding, correct Content-Range.
GET http://localhost:8080/static/style.css
Range: bytes=0-9
Accept-Encoding: gzip
HTTP 206
[Asserts]
header "Content-Range" exists
header "Content-Length" == "10"
header "Etag" not exists
header "Content-Encoding" not exists
body == "body {\n  f"

# --- GET /api/health ----------------------------------------------------

# A normal JSON endpoint. Same global ETag as /static/, no scoping
# needed either way: it never serves Range-capable content, so there's
# nothing for ETag to step aside from.
GET http://localhost:8080/api/health
HTTP 200
[Asserts]
jsonpath "$.status" == "ok"
header "Etag" exists
[Captures]
health_etag: header "Etag"

# Reusing that Etag via If-None-Match gets the expected 304, with no
# body.
GET http://localhost:8080/api/health
If-None-Match: {{health_etag}}
HTTP 304
