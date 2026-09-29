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
// doesn't link it again by title. CoverPageID is the page the tablet shows
// as its cover, and HasCover whether that page's thumbnail has been synced;
// CoverCheckedAt is the document's LastModified when the cover was last
// fetched, so it's fetched again only once the document changes.
// ParentUUID is the folder holding the document on the tablet: "" for the
// top of the library, TrashFolderUUID for the trash, or a RemarkableFolder's
// UUID.
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
	CoverPageID       string     `json:"-"`
	HasCover          bool       `json:"hasCover"`
	CoverCheckedAt    *time.Time `json:"-"`
	ParentUUID        string     `json:"parentUuid,omitempty"`
}

// TrashFolderUUID is the parent the tablet gives documents and folders in
// its trash.
const TrashFolderUUID = "trash"

// RemarkableFolder is a folder in a tablet's library, inside the folder
// ParentUUID, which is "" at the top of the library.
type RemarkableFolder struct {
	UUID       string `json:"uuid"`
	Title      string `json:"title"`
	ParentUUID string `json:"parentUuid,omitempty"`
}
