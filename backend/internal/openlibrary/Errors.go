package openlibrary

import (
	"net/http"

	"github.com/ansel1/merry"
)

var (
	ErrEmptyQuery          = merry.New("search query or subject is required").WithHTTPCode(http.StatusBadRequest).WithUserMessage("enter a title, author or ISBN, or pick a genre")
	ErrQueryTooShort       = merry.New("search query is too short").WithHTTPCode(http.StatusBadRequest).WithUserMessagef("search for at least %d characters", MinimumQueryLength)
	ErrQueryRejected       = merry.New("open library rejected the search query").WithHTTPCode(http.StatusBadRequest).WithUserMessage("Open Library can't search for that, try a more specific search")
	ErrInvalidSort         = merry.New("invalid sort").WithHTTPCode(http.StatusBadRequest).WithUserMessage("sort by relevance, rating, new or old")
	ErrInvalidLanguage     = merry.New("invalid language").WithHTTPCode(http.StatusBadRequest).WithUserMessage("language must be a 3-letter code such as eng")
	ErrInvalidSubject      = merry.New("invalid subject").WithHTTPCode(http.StatusBadRequest).WithUserMessage("genre must be letters, numbers and spaces")
	ErrInvalidPage         = merry.New("invalid page").WithHTTPCode(http.StatusBadRequest).WithUserMessage("page must be 1 or more")
	ErrInvalidYearRange    = merry.New("invalid year range").WithHTTPCode(http.StatusBadRequest).WithUserMessage("the published year range is invalid")
	ErrUpstreamUnavailable = merry.New("open library is unavailable").WithHTTPCode(http.StatusBadGateway).WithUserMessage("Open Library is unavailable, try again shortly")
	ErrEmptyISBN           = merry.New("isbn is required").WithHTTPCode(http.StatusBadRequest)
	ErrEditionNotFound     = merry.New("edition not found").WithHTTPCode(http.StatusNotFound)
)
