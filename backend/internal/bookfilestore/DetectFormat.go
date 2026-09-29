package bookfilestore

import (
	"bytes"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

// formatHeaderLength is how much of a file DetectFormat needs: an EPUB is a
// zip whose first entry, at byte 30, is the uncompressed "mimetype" file.
const formatHeaderLength = 58

var (
	pdfSignature  = []byte("%PDF-")
	zipSignature  = []byte("PK\x03\x04")
	epubMimetype  = []byte("mimetypeapplication/epub+zip")
	epubMimeStart = 30
)

// DetectFormat tells an EPUB from a PDF by the start of the file.
func DetectFormat(header []byte) (models.FileType, error) {
	switch {
	case bytes.HasPrefix(header, pdfSignature):
		return models.FileTypePDF, nil
	case bytes.HasPrefix(header, zipSignature) && len(header) >= formatHeaderLength &&
		bytes.Equal(header[epubMimeStart:formatHeaderLength], epubMimetype):
		return models.FileTypeEPUB, nil
	}
	return "", ErrUnsupportedFormat
}
