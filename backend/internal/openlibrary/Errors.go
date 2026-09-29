package openlibrary

import "errors"

var (
	ErrEmptyQuery          = errors.New("search query or subject is required")
	ErrQueryTooShort       = errors.New("search query is too short")
	ErrQueryRejected       = errors.New("open library rejected the search query")
	ErrInvalidSort         = errors.New("invalid sort")
	ErrInvalidLanguage     = errors.New("invalid language")
	ErrInvalidSubject      = errors.New("invalid subject")
	ErrInvalidPage         = errors.New("invalid page")
	ErrInvalidYearRange    = errors.New("invalid year range")
	ErrUpstreamUnavailable = errors.New("open library is unavailable")
	ErrEmptyISBN           = errors.New("isbn is required")
	ErrEditionNotFound     = errors.New("edition not found")
	ErrInvalidWorkID       = errors.New("invalid open library work id")
	ErrWorkNotFound        = errors.New("work not found")
	ErrNotPublicDomain     = errors.New("work is not in the public domain")
)
