package middleware

// CompressOption configures Compress.
type CompressOption func(*compressConfig)

type compressConfig struct {
	level int
	types []string
}

// WithCompressionLevel sets the gzip level, as defined by compress/gzip.
// Defaults to gzip.DefaultCompression.
//
// Compress panics on a level gzip rejects: the level is a constant in the
// caller's source, so a wrong one is a programming error that should surface
// at startup rather than on the first compressible response.
func WithCompressionLevel(level int) CompressOption {
	return func(config *compressConfig) {
		config.level = level
	}
}

// WithCompressibleTypes replaces the set of response Content-Type values worth
// compressing. A trailing "/*" matches any subtype, so "text/*" covers
// text/plain and text/csv alike.
//
// The default list already covers the textual types an API serves - JSON,
// problem+json, HTML, CSS, JavaScript, XML, SVG and plain text. Replace it to
// add a type of your own, not to restate those.
func WithCompressibleTypes(types ...string) CompressOption {
	return func(config *compressConfig) {
		config.types = types
	}
}
