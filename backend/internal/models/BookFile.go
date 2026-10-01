package models

import "time"

type BookFileSource string

const (
	// BookFileSourceOpenLibrary is a public domain ebook fetched through
	// Open Library from the Internet Archive.
	BookFileSourceOpenLibrary BookFileSource = "open_library"
	// BookFileSourceUpload is a file the user uploaded.
	BookFileSourceUpload BookFileSource = "upload"
	// BookFileSourceFetched is an ebook fetched from a book source, named
	// by SourceName.
	BookFileSourceFetched BookFileSource = "fetched"
)

// BookFile is an ebook kept on the server for a book, which sync copies to
// every paired tablet. Deliveries lists the tablets it has been copied to.
type BookFile struct {
	BookID     string         `json:"bookId"`
	Format     FileType       `json:"format"`
	Size       int64          `json:"size"`
	Source     BookFileSource `json:"source"`
	SourceName string         `json:"sourceName,omitempty"`
	SavedAt    time.Time      `json:"savedAt"`
	Deliveries []Delivery     `json:"deliveries"`
}

// Delivery is a book file copied to a tablet as the document DocumentUUID.
// LoadedAt is nil until the tablet's reading app restarts and shows it.
type Delivery struct {
	DeviceID     string     `json:"deviceId"`
	DocumentUUID string     `json:"documentUuid"`
	CopiedAt     time.Time  `json:"copiedAt"`
	LoadedAt     *time.Time `json:"loadedAt,omitempty"`
}
