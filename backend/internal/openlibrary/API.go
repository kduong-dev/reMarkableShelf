package openlibrary

// API is the front interface other packages depend on to search Open
// Library for candidate books to add to a user's collection. Handler.OpenLibrary
// is typed as API rather than *APIClient so it can be substituted with a fake
// in handler tests.
type API interface {
	Search(query string) ([]Result, error)
}

// Result is the subset of an Open Library work surfaced to the frontend
// for the "search and add to collection" flow.
type Result struct {
	OpenLibraryID string `json:"openLibraryId"`
	Title         string `json:"title"`
	Author        string `json:"author"`
	ISBN          string `json:"isbn,omitempty"`
	CoverURL      string `json:"coverUrl,omitempty"`
}
