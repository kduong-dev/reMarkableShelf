// Package sourceplugin is the contract between reMarkable Shelf and the
// plugins that let it fetch ebooks from other sources. A plugin is a web
// service, in any language, answering three requests:
//
//	GET  /manifest            describes the plugin, as a Manifest
//	POST /ebooks/find         takes a FindRequest, answers a FindResponse
//	GET  /ebooks/{id}/content sends the ebook's file, or 404 if it's gone
//
// with {id} path-escaped. A plugin written in Go can implement Source and
// serve it with NewHandler.
package sourceplugin

import (
	"context"
	"io"
)

// Manifest names a plugin for the Sources page.
type Manifest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// FindRequest is the book to find ebooks of. Any field may be empty; a
// plugin matches on what it can.
type FindRequest struct {
	Title         string `json:"title"`
	Author        string `json:"author,omitempty"`
	ISBN          string `json:"isbn,omitempty"`
	OpenLibraryID string `json:"openLibraryId,omitempty"`
}

type FindResponse struct {
	Ebooks []Ebook `json:"ebooks"`
}

type Format string

const (
	FormatEPUB Format = "epub"
	FormatPDF  Format = "pdf"
)

// Ebook is one file a plugin can send. ID is opaque to reMarkable Shelf,
// which only hands it back to fetch the content. Size is in bytes, 0 when
// unknown, and Description tells editions apart, such as "with images".
type Ebook struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Author      string `json:"author,omitempty"`
	Format      Format `json:"format"`
	Size        int64  `json:"size,omitempty"`
	Description string `json:"description,omitempty"`
}

// Source is a plugin's ebooks, for serving with NewHandler.
type Source interface {
	Manifest() Manifest
	FindEbooks(ctx context.Context, request FindRequest) ([]Ebook, error)
	// OpenEbook returns the ebook's content, which the caller closes, its
	// format and its size in bytes, or ErrEbookNotFound.
	OpenEbook(ctx context.Context, id string) (Content, error)
}

// Content is an ebook's file, with Size 0 when unknown.
type Content struct {
	Body   io.ReadCloser
	Format Format
	Size   int64
}
