package openlibrary

import (
	"net/http"

	"github.com/ansel1/merry"
)

var (
	ErrEmptyQuery          = merry.New("search query is required").WithHTTPCode(http.StatusBadRequest)
	ErrUpstreamUnavailable = merry.New("open library is unavailable").WithHTTPCode(http.StatusBadGateway)
	ErrEmptyISBN           = merry.New("isbn is required").WithHTTPCode(http.StatusBadRequest)
	ErrEditionNotFound     = merry.New("edition not found").WithHTTPCode(http.StatusNotFound)
)
