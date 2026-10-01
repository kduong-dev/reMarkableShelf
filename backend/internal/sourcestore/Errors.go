package sourcestore

import "errors"

var (
	ErrSourceNotFound   = errors.New("ebook source not found")
	ErrBuiltInSource    = errors.New("the built-in ebook source can't be removed")
	ErrSourceExists     = errors.New("ebook source already added")
	ErrInvalidOrder     = errors.New("the order must list every ebook source once")
	ErrSourceNameNeeded = errors.New("ebook source needs a name")
)
