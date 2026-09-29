package openlibrary

// API is the front interface other packages depend on to search Open
// Library for candidate books to add to a user's collection. Handler.OpenLibrary
// is typed as API rather than *APIClient so it can be substituted with a fake
// in handler tests.
type API interface {
	Search(input SearchInput) (SearchResults, error)
	// EditionPageCount returns the page count of the edition with the given
	// ISBN, or 0 when Open Library doesn't record one.
	EditionPageCount(isbn string) (int, error)
	// PublicScans returns the Internet Archive identifiers of the work's
	// scans, for a work in the public domain, and ErrNotPublicDomain
	// otherwise. Some of the identifiers may still only be borrowable.
	PublicScans(openLibraryID string) (Scans, error)
}

// Scans is a public-domain work's title and the Internet Archive identifiers
// of its scans.
type Scans struct {
	Title      string
	ArchiveIDs []string
}

// SearchResults is one page of a search. Total counts every match, so the
// caller can tell whether more pages remain.
type SearchResults struct {
	Results []Result `json:"results"`
	Total   int      `json:"total"`
}

// Result is the subset of an Open Library work surfaced to the frontend
// for the "search and add to collection" flow. PageCount is the median across
// the work's editions, so it may differ from the edition the user owns, and
// AverageRating is the average of Open Library readers' 1-5 star ratings.
type Result struct {
	OpenLibraryID    string  `json:"openLibraryId"`
	Title            string  `json:"title"`
	Author           string  `json:"author"`
	ISBN             string  `json:"isbn,omitempty"`
	CoverURL         string  `json:"coverUrl,omitempty"`
	PageCount        int     `json:"pageCount,omitempty"`
	FirstPublishYear int     `json:"firstPublishYear,omitempty"`
	AverageRating    float64 `json:"averageRating,omitempty"`
	// Downloadable is true when the work is in the public domain, so a free
	// scan of it can be downloaded.
	Downloadable bool `json:"downloadable,omitempty"`
}
