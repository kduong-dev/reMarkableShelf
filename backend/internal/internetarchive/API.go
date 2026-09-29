package internetarchive

import (
	"context"
	"io"
)

// API is the front interface other packages depend on to download public
// domain ebooks from the Internet Archive, so it can be substituted with a
// fake in handler tests.
type API interface {
	// FindEbook picks an ebook among the scans with the given identifiers,
	// leaving out scans that can only be borrowed. It prefers EPUB, falling
	// back to PDF, and returns ErrNoPublicEbook when none has either.
	FindEbook(identifiers []string) (Ebook, error)
	// OpenEbook starts downloading the ebook. The caller closes the
	// Download's Body.
	OpenEbook(ctx context.Context, ebook Ebook) (Download, error)
}

type Format string

const (
	FormatEPUB Format = "epub"
	FormatPDF  Format = "pdf"
)

// Ebook is a file of an Internet Archive item.
type Ebook struct {
	Identifier string
	FileName   string
	Format     Format
}

// Download is an ebook's contents, with ContentLength -1 when the archive
// doesn't say how long it is.
type Download struct {
	Body          io.ReadCloser
	ContentLength int64
}
