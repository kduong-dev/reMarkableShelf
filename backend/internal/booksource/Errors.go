package booksource

import "errors"

var (
	ErrNoEbook           = errors.New("no source has an ebook of the book")
	ErrEbookNotFound     = errors.New("the source no longer has that ebook")
	ErrInvalidEbookID    = errors.New("invalid ebook id")
	ErrSourceUnavailable = errors.New("ebook source is unavailable")
	ErrUnknownKind       = errors.New("unknown kind of ebook source")
	ErrInvalidURL        = errors.New("ebook source URL must be http or https")
	ErrNotPlugin         = errors.New("no ebook source plugin answers at that URL")
	ErrNotOPDS           = errors.New("no OPDS catalog at that URL")
	ErrNotSearchable     = errors.New("the OPDS catalog can't be searched")
)
