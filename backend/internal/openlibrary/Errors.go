package openlibrary

import (
	"net/http"

	"github.com/ansel1/merry"
)

var (
	ErrEmptyQuery          = merry.New("search query is required").WithHTTPCode(http.StatusBadRequest)
	ErrUpstreamUnavailable = merry.New("open library is unavailable").WithHTTPCode(http.StatusBadGateway)
)
