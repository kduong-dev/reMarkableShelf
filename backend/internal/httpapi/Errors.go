package httpapi

import (
	"net/http"

	"github.com/ansel1/merry"
)

var (
	ErrInvalidNumber     = merry.New("invalid number").WithHTTPCode(http.StatusBadRequest)
	ErrFileTooLarge      = merry.New("file too large").WithHTTPCode(http.StatusRequestEntityTooLarge)
	ErrNoOpenLibraryWork = merry.New("book isn't matched to an open library work").
				WithHTTPCode(http.StatusBadRequest).
				WithUserMessage("find this book on Open Library first, so its free ebook can be looked up")
)
