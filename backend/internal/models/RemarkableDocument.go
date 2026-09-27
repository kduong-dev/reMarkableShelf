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
// CurrentPage counts from 1 in the tablet's own pages and is nil for a
// document never opened; PositionUpdatedAt is the tablet's last activity
// on the document, used to tell whether its position is newer than the app's.
// BookTitle and BookAuthor are read from the file itself when the tablet
// could (usually EPUBs), whereas Title is its name in the tablet's library.
// AutoLinkDismissed records that the user unlinked the document, so sync
// doesn't link it again by title.
type RemarkableDocument struct {
	UUID              string     `json:"uuid"`
	DeviceID          string     `json:"deviceId"`
	Title             string     `json:"title"`
	FileType          FileType   `json:"fileType"`
	LastModified      time.Time  `json:"lastModified"`
	LinkedBookID      *string    `json:"linkedBookId,omitempty"`
	CurrentPage       *int       `json:"currentPage,omitempty"`
	PageCount         *int       `json:"pageCount,omitempty"`
	PositionUpdatedAt *time.Time `json:"positionUpdatedAt,omitempty"`
	BookTitle         string     `json:"bookTitle,omitempty"`
	BookAuthor        string     `json:"bookAuthor,omitempty"`
	AutoLinkDismissed bool       `json:"-"`
}
