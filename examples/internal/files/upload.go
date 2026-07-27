package files

import (
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"

	"github.com/Casara/arnon/httpx"
	"github.com/Casara/arnon/problem"
)

const (
	// maxUploadMemory bounds how much of the request body
	// ParseMultipartForm keeps in memory before spilling the rest to
	// a temporary file - it is not a second size limit, the overall
	// body is already capped upstream by middleware.MaxBodyBytes
	// (examples/cmd/files/main.go). net/http removes any spilled temp
	// file automatically once the handler returns (confirmed in
	// net/http's response.finishRequest, which calls
	// req.MultipartForm.RemoveAll()), so Upload doesn't need to.
	maxUploadMemory = 1 << 20 // 1 MiB

	sniffLength = 512
)

// UploadSummary is the JSON body Upload answers with. A plain
// http.Handler is free to respond in JSON even though its request
// wasn't - httpx.Endpoint's JSON-only rule is about the typed binding
// pipeline, not about what a handler is allowed to write.
type UploadSummary struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Bytes       int64  `json:"bytes"`
	Elements    int    `json:"elements,omitempty"`
}

// Upload accepts one file (multipart field "file") and validates it
// in two layers before answering with an UploadSummary:
//
//  1. The Content-Type the client declared for that part must be one
//     of text/csv, application/xml, text/xml, or application/pdf.
//     middleware.AllowContentType can't do this check by itself: it
//     only sees the envelope's own Content-Type
//     (multipart/form-data; boundary=...), never the type of an
//     individual part inside it.
//  2. The file is then actually parsed as whatever it claims to be
//     (csv.Reader, xml.Decoder token-by-token, or a PDF magic-byte
//     sniff), so a mislabeled part is caught by content, not just by
//     its declared header. PDF is checked by sniffing
//     (http.DetectContentType) because a real parse is out of scope
//     here; csv/xml are instead checked by parsing them for real,
//     which needs no sniffing step and - unlike sniffing - has no
//     false-positive risk from something like an optional XML
//     prolog (confirmed empirically: a well-formed XML document with
//     no leading "<?xml" declaration sniffs as plain text, which a
//     sniff-based check would wrongly reject; encoding/xml has no
//     such blind spot, since it parses structure rather than
//     matching a byte prefix).
func Upload(
	writer http.ResponseWriter,
	request *http.Request,
) {
	//nolint:gosec // the request body as a whole (not just what stays
	// in memory here) is already bounded by middleware.MaxBodyBytes,
	// applied to the /uploads group in examples/cmd/files/main.go -
	// gosec can't see across that middleware boundary from here.
	err := request.ParseMultipartForm(maxUploadMemory)
	if err != nil {
		writeUploadProblem(
			writer, request,
			http.StatusBadRequest, "Invalid multipart body", err.Error(),
		)

		return
	}

	file, header, err := request.FormFile("file")
	if err != nil {
		writeUploadProblem(
			writer, request,
			http.StatusBadRequest, "Missing file",
			`expected a "file" field in the multipart body`,
		)

		return
	}
	defer func() { _ = file.Close() }()

	declaredType, _, _ := mime.ParseMediaType(header.Header.Get("Content-Type"))

	if !isAllowedUploadType(declaredType) {
		writeUploadProblem(
			writer, request,
			http.StatusUnsupportedMediaType, "Unsupported file type",
			fmt.Sprintf(
				"content type %q is not one of the accepted types (text/csv, application/xml, text/xml, application/pdf)",
				declaredType,
			),
		)

		return
	}

	summary := UploadSummary{
		Filename:    header.Filename,
		ContentType: declaredType,
		Bytes:       header.Size,
		// Elements is filled in below, per content type; a PDF has
		// none.
		Elements: 0,
	}

	switch declaredType {
	case "text/csv":
		rows, readErr := csv.NewReader(file).ReadAll()
		if readErr != nil {
			writeUploadProblem(
				writer,
				request,
				http.StatusBadRequest,
				"Invalid CSV",
				readErr.Error(),
			)

			return
		}

		summary.Elements = len(rows)
	case "application/xml", "text/xml":
		count, decodeErr := countXMLElements(file)
		if decodeErr != nil {
			writeUploadProblem(
				writer,
				request,
				http.StatusBadRequest,
				"Invalid XML",
				decodeErr.Error(),
			)

			return
		}

		summary.Elements = count
	default: // application/pdf
		matches, sniffedType, sniffErr := pdfSignatureMatches(file)
		if sniffErr != nil {
			writeUploadProblem(
				writer,
				request,
				http.StatusBadRequest,
				"Unreadable file",
				sniffErr.Error(),
			)

			return
		}

		if !matches {
			writeUploadProblem(
				writer, request,
				http.StatusUnprocessableEntity, "Content mismatch",
				fmt.Sprintf(
					"declared content type %q does not match the file's actual content (%q)",
					declaredType, sniffedType,
				),
			)

			return
		}
	}

	_ = httpx.WriteJSON(writer, http.StatusOK, summary)
}

func isAllowedUploadType(contentType string) bool {
	switch contentType {
	case "text/csv", "application/xml", "text/xml", "application/pdf":
		return true
	default:
		return false
	}
}

// pdfSignatureMatches sniffs the first sniffLength bytes and reports
// whether they match PDF's magic bytes ("%PDF-", the same signature
// http.DetectContentType itself checks - see net/http/sniff.go),
// along with whatever type was actually sniffed. It's only meaningful
// for PDF: unlike XML's optional "<?xml" prolog, a PDF's header is
// mandatory and unambiguous, so this can't produce the false positive
// a prolog-less-XML sniff would.
func pdfSignatureMatches(file io.Reader) (bool, string, error) {
	buffer := make([]byte, sniffLength)

	n, err := io.ReadFull(file, buffer)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return false, "", fmt.Errorf("read file for sniffing: %w", err)
	}

	sniffedType, _, _ := mime.ParseMediaType(http.DetectContentType(buffer[:n]))

	return sniffedType == "application/pdf", sniffedType, nil
}

// countXMLElements decodes the document token by token, verifying
// it's well-formed XML without loading it into a DOM/struct, and
// returns the number of start elements found.
func countXMLElements(reader io.Reader) (int, error) {
	decoder := xml.NewDecoder(reader)

	count := 0

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return 0, fmt.Errorf("decode xml: %w", err)
		}

		if _, ok := token.(xml.StartElement); ok {
			count++
		}
	}

	return count, nil
}

func writeUploadProblem(
	writer http.ResponseWriter,
	request *http.Request,
	status int,
	title string,
	detail string,
) {
	httpx.WriteProblem(writer, request, problem.New(status, title, detail))
}
