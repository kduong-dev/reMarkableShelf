package models

import "time"

type EbookSourceKind string

const (
	// EbookSourceKindInternetArchive is the built-in source of public domain
	// scans, found through Open Library.
	EbookSourceKindInternetArchive EbookSourceKind = "internet_archive"
	// EbookSourceKindPlugin is a web service answering the sourceplugin
	// contract at URL.
	EbookSourceKindPlugin EbookSourceKind = "plugin"
	// EbookSourceKindOPDS is an OPDS catalog whose root feed is at URL.
	EbookSourceKindOPDS EbookSourceKind = "opds"
)

// BuiltInEbookSourceID is the Internet Archive source every library has,
// which can be disabled or reordered but not removed.
const BuiltInEbookSourceID = "internet-archive"

// EbookSource is where ebooks can be fetched from. Sources are tried in
// Priority order, lowest first, skipping disabled ones. SearchURL is an OPDS
// catalog's search template, found when it was added.
type EbookSource struct {
	ID        string          `json:"id"`
	Kind      EbookSourceKind `json:"kind"`
	Name      string          `json:"name"`
	URL       string          `json:"url,omitempty"`
	SearchURL string          `json:"-"`
	Priority  int             `json:"priority"`
	Enabled   bool            `json:"enabled"`
	CreatedAt time.Time       `json:"createdAt"`
}
