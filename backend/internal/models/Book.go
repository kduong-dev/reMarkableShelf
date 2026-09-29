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

type ProgressSource string

const (
	ProgressSourceApp        ProgressSource = "app"
	ProgressSourceRemarkable ProgressSource = "remarkable"
)

// Book is an entry in the user's tracked collection. It may originate from a
// manual search-and-add (SourceManual) or be created automatically when a
// synced PDF/EPUB from a tablet couldn't be matched to an existing entry
// (SourceRemarkable). ProgressUpdatedAt and ProgressSource record when the
// bookmark last moved and whether the app or a tablet sync moved it.
// TabletCoverURL serves the cover of a linked tablet document, if synced,
// and TabletPageCount is its page count, which a linked book follows.
// TabletDevices names the tablets holding a document linked to the book.
type Book struct {
	ID                string         `json:"id"`
	Title             string         `json:"title"`
	Author            string         `json:"author"`
	ISBN              string         `json:"isbn,omitempty"`
	CoverURL          string         `json:"coverUrl,omitempty"`
	Status            BookStatus     `json:"status"`
	Rating            *int           `json:"rating,omitempty"`
	Source            BookSource     `json:"source"`
	OpenLibraryID     string         `json:"openLibraryId,omitempty"`
	PageCount         *int           `json:"pageCount,omitempty"`
	CurrentPage       *int           `json:"currentPage,omitempty"`
	ProgressUpdatedAt *time.Time     `json:"progressUpdatedAt,omitempty"`
	ProgressSource    ProgressSource `json:"progressSource,omitempty"`
	TabletCoverURL    string         `json:"tabletCoverUrl,omitempty"`
	TabletPageCount   *int           `json:"tabletPageCount,omitempty"`
	TabletDevices     []string       `json:"tabletDevices,omitempty"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}
