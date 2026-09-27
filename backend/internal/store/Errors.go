package store

import (
	"net/http"

	"github.com/ansel1/merry"
)

var (
	ErrInvalidPageCount      = merry.New("page count must be at least 1").WithHTTPCode(http.StatusBadRequest)
	ErrCurrentPageOutOfRange = merry.New("current page is out of range").WithHTTPCode(http.StatusBadRequest)
)
