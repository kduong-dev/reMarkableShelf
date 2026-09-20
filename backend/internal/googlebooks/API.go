package googlebooks

// API is the front interface other packages depend on to search Google
// Books for candidate books to add to a user's collection. Handler.GoogleBooks
// is typed as API rather than *Client so it can be substituted with a fake
// in handler tests.
type API interface {
	Search(query string) ([]Result, error)
}

// Result is the subset of a Google Books volume surfaced to the frontend
// for the "search and add to collection" flow.
type Result struct {
	GoogleBooksID string `json:"googleBooksId"`
	Title         string `json:"title"`
	Author        string `json:"author"`
	ISBN          string `json:"isbn,omitempty"`
	CoverURL      string `json:"coverUrl,omitempty"`
}
