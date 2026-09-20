package models

import "time"

type FileType string

const (
	// FileTypePDF and FileTypeEPUB are imported documents synced from a
	// reMarkable tablet — these are treated as candidate books.
	FileTypePDF  FileType = "pdf"
	FileTypeEPUB FileType = "epub"
	// FileTypeNotebook is a native handwritten notebook/sketch with no
	// source document — never treated as a book, shown as a Note instead.
	FileTypeNotebook FileType = "notebook"
)

// RemarkableDocument mirrors one document (xochitl UUID) found on a synced
// tablet. PDFs/EPUBs can be linked to a Book in the collection; notebooks
// are kept separate but may optionally reference a Book (reading notes).
type RemarkableDocument struct {
	UUID         string    `json:"uuid"`
	DeviceID     string    `json:"deviceId"`
	Title        string    `json:"title"`
	FileType     FileType  `json:"fileType"`
	LastModified time.Time `json:"lastModified"`
	LinkedBookID *string   `json:"linkedBookId,omitempty"`
}
