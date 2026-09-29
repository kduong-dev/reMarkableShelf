package openlibrary

import (
	"fmt"
	"math"
	"strings"
)

const coverURLFormat = "https://covers.openlibrary.org/b/id/%d-M.jpg"

// searchResponse mirrors the subset of the Open Library search response
// (https://openlibrary.org/dev/docs/api/search) needed to build a Result.
type searchResponse struct {
	NumFound int            `json:"numFound"`
	Docs     []workDocument `json:"docs"`
}

type workDocument struct {
	Key              string   `json:"key"`
	Title            string   `json:"title"`
	AuthorName       []string `json:"author_name"`
	ISBN             []string `json:"isbn"`
	CoverID          int      `json:"cover_i"`
	PageCount        int      `json:"number_of_pages_median"`
	FirstPublishYear int      `json:"first_publish_year"`
	RatingsAverage   float64  `json:"ratings_average"`
	EbookAccess      string   `json:"ebook_access"`
	ArchiveIDs       []string `json:"ia"`
}

// ebookAccessPublic is the ebook_access of a work with a public-domain scan,
// as opposed to one that can only be borrowed.
const ebookAccessPublic = "public"

// isPublicDomain reports whether any of the work's scans is free to download.
// Search leaves out the scans themselves, as a work can have hundreds.
func (document workDocument) isPublicDomain() bool {
	return document.EbookAccess == ebookAccessPublic
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
		OpenLibraryID:    strings.TrimPrefix(document.Key, "/works/"),
		Title:            document.Title,
		Author:           author,
		ISBN:             isbn,
		CoverURL:         coverURL,
		PageCount:        document.PageCount,
		FirstPublishYear: document.FirstPublishYear,
		AverageRating:    math.Round(document.RatingsAverage*10) / 10,
		Downloadable:     document.isPublicDomain(),
	}
}
