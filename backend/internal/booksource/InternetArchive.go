package booksource

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/internetarchive"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
)

// internetArchive finds public domain scans of a book matched to Open
// Library. Its ebook IDs are "<identifier>/<file name>".
type internetArchive struct {
	openLibrary     openlibrary.API
	internetArchive internetarchive.API
}

func (source internetArchive) FindEbooks(ctx context.Context, query Query) ([]Ebook, error) {
	if query.OpenLibraryID == "" {
		return nil, nil
	}
	scans, err := source.openLibrary.PublicScans(query.OpenLibraryID)
	if errors.Is(err, openlibrary.ErrNotPublicDomain) || errors.Is(err, openlibrary.ErrWorkNotFound) || errors.Is(err, openlibrary.ErrInvalidWorkID) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSourceUnavailable, err)
	}
	ebook, err := source.internetArchive.FindEbook(scans.ArchiveIDs)
	if errors.Is(err, internetarchive.ErrNoPublicEbook) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSourceUnavailable, err)
	}
	return []Ebook{{
		ID:          ebook.Identifier + "/" + ebook.FileName,
		Title:       scans.Title,
		Format:      models.FileType(ebook.Format),
		Description: "Internet Archive item " + ebook.Identifier,
	}}, nil
}

func (source internetArchive) OpenEbook(ctx context.Context, ebookID string) (Download, error) {
	identifier, fileName, ok := strings.Cut(ebookID, "/")
	if !ok || identifier == "" || fileName == "" {
		return Download{}, fmt.Errorf("%w %q", ErrInvalidEbookID, ebookID)
	}
	download, err := source.internetArchive.OpenEbook(ctx, internetarchive.Ebook{Identifier: identifier, FileName: fileName})
	if err != nil {
		return Download{}, fmt.Errorf("%w: %w", ErrSourceUnavailable, err)
	}
	return Download{Body: download.Body, ContentLength: download.ContentLength}, nil
}
