package booksource

import (
	"context"
	"io"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

// Source is somewhere ebooks of a book can be fetched from: the built-in
// Internet Archive, a plugin, or an OPDS catalog.
type Source interface {
	// FindEbooks lists the source's EPUBs and PDFs of the book, best first,
	// and none, without an error, when it has none.
	FindEbooks(ctx context.Context, query Query) ([]Ebook, error)
	// OpenEbook starts fetching an ebook FindEbooks listed. The caller
	// closes the Download's Body.
	OpenEbook(ctx context.Context, ebookID string) (Download, error)
}

// Query is the book to find ebooks of; any field may be empty.
type Query struct {
	Title         string
	Author        string
	ISBN          string
	OpenLibraryID string
}

// Ebook is a file a source can fetch. ID is the source's own, handed back
// to OpenEbook. Size is in bytes, 0 when unknown, and Description tells
// editions apart.
type Ebook struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Author      string          `json:"author,omitempty"`
	Format      models.FileType `json:"format"`
	Size        int64           `json:"size,omitempty"`
	Description string          `json:"description,omitempty"`
}

// Download is an ebook's contents, with ContentLength -1 when unknown.
type Download struct {
	Body          io.ReadCloser
	ContentLength int64
}
