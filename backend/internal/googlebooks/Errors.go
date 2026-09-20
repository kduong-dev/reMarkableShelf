package googlebooks

import (
	"net/http"

	"github.com/ansel1/merry"
)

var (
	ErrEmptyQuery          = merry.New("search query is required").WithHTTPCode(http.StatusBadRequest)
	ErrUpstreamUnavailable = merry.New("google books is unavailable").WithHTTPCode(http.StatusBadGateway)
)
