package openlibrary

// editionResponse mirrors the subset of an Open Library edition
// (https://openlibrary.org/dev/docs/api/books) needed for its page count.
type editionResponse struct {
	PageCount int `json:"number_of_pages"`
}
