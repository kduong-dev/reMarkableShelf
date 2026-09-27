package openlibrary

import (
	"fmt"
	"strings"
)

const coverURLFormat = "https://covers.openlibrary.org/b/id/%d-M.jpg"

// searchResponse mirrors the subset of the Open Library search response
// (https://openlibrary.org/dev/docs/api/search) needed to build a Result.
type searchResponse struct {
	Docs []workDocument `json:"docs"`
}

type workDocument struct {
	Key        string   `json:"key"`
	Title      string   `json:"title"`
	AuthorName []string `json:"author_name"`
	ISBN       []string `json:"isbn"`
	CoverID    int      `json:"cover_i"`
}

// toResult picks the fields the frontend needs out of a work. A work lists
// the ISBNs of all its editions, so the first 13-digit ISBN is preferred.
func (document workDocument) toResult() Result {
	author := ""
	if len(document.AuthorName) > 0 {
		author = document.AuthorName[0]
	}
	isbn := ""
	for _, candidate := range document.ISBN {
		if len(candidate) == 13 {
			isbn = candidate
			break
		}
		if isbn == "" {
			isbn = candidate
		}
	}
	coverURL := ""
	if document.CoverID > 0 {
		coverURL = fmt.Sprintf(coverURLFormat, document.CoverID)
	}
	return Result{
		OpenLibraryID: strings.TrimPrefix(document.Key, "/works/"),
		Title:         document.Title,
		Author:        author,
		ISBN:          isbn,
		CoverURL:      coverURL,
	}
}
