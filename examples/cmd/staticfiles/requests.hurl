# Requests for examples/cmd/staticfiles.
#
# Run with the example server up (`go run ./examples/cmd/staticfiles`):
#
#   hurl --test examples/cmd/staticfiles/requests.hurl

# --- GET /static/ -----------------------------------------------------

# http.FileServer serves assets/index.html for the directory root.
# No Etag here: /static/ deliberately isn't wrapped in arnon's ETag -
# see the doc comment in main.go for why.
GET http://localhost:8080/static/
HTTP 200
[Asserts]
header "Content-Type" == "text/html; charset=utf-8"
header "Accept-Ranges" == "bytes"
header "Etag" not exists

# /static/index.html itself redirects to /static/ - http.FileServer's
# own behavior for a file literally named "index.html", nothing
# arnon-specific.
GET http://localhost:8080/static/index.html
HTTP 301
[Asserts]
header "Location" == "./"

# --- GET /static/style.css ---------------------------------------------

# A Range request against a static asset: http.FileServer's own Range
# support, untouched by arnon (no ETag/Compress on this route).
GET http://localhost:8080/static/style.css
HTTP 200
[Asserts]
header "Etag" not exists

GET http://localhost:8080/static/style.css
Range: bytes=0-9
HTTP 206
[Asserts]
header "Content-Range" exists
header "Content-Length" == "10"
header "Etag" not exists

# --- GET /api/health ----------------------------------------------------

# A normal JSON endpoint, wrapped in arnon's ETag (safe here: it never
# serves Range-capable content).
GET http://localhost:8080/api/health
HTTP 200
[Asserts]
jsonpath "$.status" == "ok"
header "Etag" exists
[Captures]
health_etag: header "Etag"

# Reusing that Etag via If-None-Match gets the expected 304, with no
# body - the conditional-GET case ETag exists for, and the one /static/
# gets for free from http.FileServer instead (via Last-Modified).
GET http://localhost:8080/api/health
If-None-Match: {{health_etag}}
HTTP 304
