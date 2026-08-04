// Package files provides the shared file download/upload handlers
// used by examples/cmd/files: plain http.Handlers demonstrating
// non-JSON content, which httpx.Endpoint deliberately doesn't cover
// (it's JSON-only by design - see httpx/CLAUDE.md). None of these routes
// bind, sanitize, or validate anything; they're mounted directly on
// the router, exactly like Endpoint's own doc comment says any other
// representation should be.
package files

import (
	"encoding/csv"
	"encoding/xml"
	"net/http"
	"strconv"

	_ "embed"
)

//go:embed testdata/sample.pdf
var samplePDF []byte

// user is the sample record shared by DownloadCSV and DownloadXML.
type user struct {
	ID    string `xml:"id"`
	Name  string `xml:"name"`
	Email string `xml:"email"`
}

func sampleUsers() []user {
	return []user{
		{ID: "usr_1", Name: "Ada Lovelace", Email: "ada@example.com"},
		{ID: "usr_2", Name: "Alan Turing", Email: "alan@example.com"},
	}
}

// DownloadPDF serves a static, pre-existing PDF asset.
// Content-Disposition marks it as an attachment, so a browser
// downloads it instead of trying to render it inline (dropping the
// header, or setting it to "inline", is the only change needed for
// that case). Content-Length is set explicitly because the asset's
// size is already known in full - net/http only fills it in
// automatically when a handler's entire response fits inside its
// internal write buffer before the first flush, which a 2 KB+ PDF
// does not.
func DownloadPDF(
	writer http.ResponseWriter,
	_ *http.Request,
) {
	writer.Header().Set("Content-Type", "application/pdf")
	writer.Header().Set("Content-Disposition", `attachment; filename="report.pdf"`)
	writer.Header().Set("Content-Length", strconv.Itoa(len(samplePDF)))

	_, _ = writer.Write(samplePDF)
}

// DownloadCSV streams a CSV file generated on the fly: csv.Writer
// encodes straight into writer as each record is produced, so -
// unlike httpx.WriteJSON, which buffers the whole body first because
// json.Marshal can fail mid-way - there's nothing to buffer for, and
// no Content-Length to compute ahead of time (the response is sent
// chunked).
func DownloadCSV(
	writer http.ResponseWriter,
	_ *http.Request,
) {
	writer.Header().Set("Content-Type", "text/csv; charset=utf-8")
	writer.Header().Set("Content-Disposition", `attachment; filename="users.csv"`)

	csvWriter := csv.NewWriter(writer)

	_ = csvWriter.Write([]string{"id", "name", "email"})

	for _, sampleUser := range sampleUsers() {
		_ = csvWriter.Write([]string{sampleUser.ID, sampleUser.Name, sampleUser.Email})
	}

	csvWriter.Flush()
}

// usersDocument is the root element wrapping DownloadXML's streamed
// output.
type usersDocument struct {
	XMLName xml.Name `xml:"users"`
	Users   []user   `xml:"user"`
}

// DownloadXML streams an XML file generated on the fly - same
// streaming rationale as DownloadCSV.
func DownloadXML(
	writer http.ResponseWriter,
	_ *http.Request,
) {
	writer.Header().Set("Content-Type", "application/xml; charset=utf-8")
	writer.Header().Set("Content-Disposition", `attachment; filename="users.xml"`)

	_, _ = writer.Write([]byte(xml.Header))

	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")

	_ = encoder.Encode(usersDocument{
		// Local mirrors the "users" xml tag on this field, which is
		// what actually fixes the marshaled root element name -
		// Space stays empty, there's no XML namespace here.
		XMLName: xml.Name{Space: "", Local: "users"},
		Users:   sampleUsers(),
	})
}
