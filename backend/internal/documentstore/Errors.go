package documentstore

import "errors"

var (
	ErrDocumentNotFound = errors.New("document not found")
	ErrCoverNotFound    = errors.New("cover not found")
)
