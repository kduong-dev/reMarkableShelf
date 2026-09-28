package bookstore

import "errors"

var (
	ErrBookNotFound          = errors.New("book not found")
	ErrInvalidPageCount      = errors.New("page count must be at least 1")
	ErrCurrentPageOutOfRange = errors.New("current page is out of range")
)
