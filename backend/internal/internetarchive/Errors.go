package internetarchive

import "errors"

var (
	ErrNoPublicEbook       = errors.New("no public ebook among the scans")
	ErrUpstreamUnavailable = errors.New("internet archive is unavailable")
)
