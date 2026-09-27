package models

import "time"

type BookStatus string

const (
	StatusWantToRead BookStatus = "want_to_read"
	StatusReading    BookStatus = "reading"
	StatusFinished   BookStatus = "finished"
)

type BookSource string

const (
	SourceManual     BookSource = "manual"
	SourceRemarkable BookSource = "remarkable"
)

// Book is an entry in the user's tracked collection. It may originate from a
// manual search-and-add (SourceManual) or be created automatically when a
// synced PDF/EPUB from a tablet couldn't be matched to an existing entry
// (SourceRemarkable).
type Book struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Author        string     `json:"author"`
	ISBN          string     `json:"isbn,omitempty"`
	CoverURL      string     `json:"coverUrl,omitempty"`
	Status        BookStatus `json:"status"`
	Rating        *int       `json:"rating,omitempty"`
	Source        BookSource `json:"source"`
	OpenLibraryID string     `json:"openLibraryId,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}
