# Requests for examples/cmd/files.
#
# Run with the example server up (`go run ./examples/cmd/files`):
#
#   hurl --test examples/cmd/files/requests.hurl

# --- GET /downloads/report.pdf --------------------------------------------

# Static asset download: Content-Length is set explicitly (the whole
# asset's size is known upfront), and Content-Disposition marks it as
# an attachment.
GET http://localhost:8080/downloads/report.pdf
HTTP 200
[Asserts]
header "Content-Type" == "application/pdf"
header "Content-Disposition" == "attachment; filename=\"report.pdf\""
header "Content-Length" == "2360"
bytes startsWith hex,255044462d31; # "%PDF-1"

# --- GET /downloads/users.csv ----------------------------------------------

# Streamed, generated on the fly - no Content-Length asserted here on
# purpose, since encoding/csv writes straight to the response as each
# record is produced.
GET http://localhost:8080/downloads/users.csv
HTTP 200
[Asserts]
header "Content-Type" == "text/csv; charset=utf-8"
header "Content-Disposition" == "attachment; filename=\"users.csv\""
body == "id,name,email\nusr_1,Ada Lovelace,ada@example.com\nusr_2,Alan Turing,alan@example.com\n"

# --- GET /downloads/users.xml ----------------------------------------------

GET http://localhost:8080/downloads/users.xml
HTTP 200
[Asserts]
header "Content-Type" == "application/xml; charset=utf-8"
header "Content-Disposition" == "attachment; filename=\"users.xml\""
xpath "string(//users/user[1]/id)" == "usr_1"
xpath "string(//users/user[2]/email)" == "alan@example.com"

# --- POST /uploads/file -----------------------------------------------

# Valid CSV.
POST http://localhost:8080/uploads/file
[Multipart]
file: file,testdata/sample.csv; text/csv
HTTP 200
[Asserts]
jsonpath "$.filename" == "sample.csv"
jsonpath "$.contentType" == "text/csv"
jsonpath "$.elements" == 3

# Valid XML.
POST http://localhost:8080/uploads/file
[Multipart]
file: file,testdata/sample.xml; application/xml
HTTP 200
[Asserts]
jsonpath "$.contentType" == "application/xml"
# 5 start elements: users, 2x user, 2x id.
jsonpath "$.elements" == 5

# Valid PDF.
POST http://localhost:8080/uploads/file
[Multipart]
file: file,testdata/sample.pdf; application/pdf
HTTP 200
[Asserts]
jsonpath "$.contentType" == "application/pdf"
jsonpath "$.bytes" == 2360

# Declared type not in the allow-list: rejected before any parsing.
POST http://localhost:8080/uploads/file
[Multipart]
file: file,testdata/sample.csv; text/plain
HTTP 415
[Asserts]
jsonpath "$.title" == "Unsupported file type"

# Missing the "file" field entirely.
POST http://localhost:8080/uploads/file
[Multipart]
notfile: file,testdata/sample.csv; text/csv
HTTP 400
[Asserts]
jsonpath "$.title" == "Missing file"

# Malformed CSV (an unterminated quoted field): declared type is
# accepted, but the actual parse fails.
POST http://localhost:8080/uploads/file
[Multipart]
file: file,testdata/broken.csv; text/csv
HTTP 400
[Asserts]
jsonpath "$.title" == "Invalid CSV"

# Mislabeled file: a CSV declared as application/pdf. PDF has no real
# parse step here, so this is caught by sniffing its magic bytes
# instead - see pdfSignatureMatches in examples/internal/files/upload.go.
POST http://localhost:8080/uploads/file
[Multipart]
file: file,testdata/sample.csv; application/pdf
HTTP 422
[Asserts]
jsonpath "$.title" == "Content mismatch"

# Mislabeled the other way: a PDF declared as XML. There's no sniffing
# step for XML (see upload.go's doc comment on why), so this is caught
# by xml.Decoder itself failing to parse binary PDF bytes as XML.
POST http://localhost:8080/uploads/file
[Multipart]
file: file,testdata/sample.pdf; application/xml
HTTP 400
[Asserts]
jsonpath "$.title" == "Invalid XML"
