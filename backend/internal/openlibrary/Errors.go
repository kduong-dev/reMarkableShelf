package openlibrary

import (
	"net/http"

	"github.com/ansel1/merry"
)

var (
	ErrEmptyQuery          = merry.New("search query is required").WithHTTPCode(http.StatusBadRequest).WithUserMessage("enter a title, author or ISBN to search")
	ErrQueryTooShort       = merry.New("search query is too short").WithHTTPCode(http.StatusBadRequest).WithUserMessagef("search for at least %d characters", MinimumQueryLength)
	ErrQueryRejected       = merry.New("open library rejected the search query").WithHTTPCode(http.StatusBadRequest).WithUserMessage("Open Library can't search for that, try a more specific search")
	ErrUpstreamUnavailable = merry.New("open library is unavailable").WithHTTPCode(http.StatusBadGateway).WithUserMessage("Open Library is unavailable, try again shortly")
	ErrEmptyISBN           = merry.New("isbn is required").WithHTTPCode(http.StatusBadRequest)
	ErrEditionNotFound     = merry.New("edition not found").WithHTTPCode(http.StatusNotFound)
)
