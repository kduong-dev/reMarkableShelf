package bookfilestore

import "errors"

var (
	ErrFileNotFound       = errors.New("book has no file")
	ErrUnsupportedFormat  = errors.New("file is not an EPUB or PDF")
	ErrStorageUnavailable = errors.New("storage service is unavailable")
)
