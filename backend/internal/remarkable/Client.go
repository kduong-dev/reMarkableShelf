package remarkable

import (
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"golang.org/x/crypto/ssh"
)

// Config is how the server signs in to tablets: as User on Port, with
// Signer, its own key, which pairing installs on each tablet. DocumentsDir
// defaults to DefaultDocumentsDir.
type Config struct {
	User         string
	Port         string
	Signer       ssh.Signer
	DocumentsDir string
}

// Tablet is where to reach a tablet and the SSH host key it identified
// itself with when paired, in authorized_keys format. Connections refuse a
// tablet presenting any other key; an empty HostKey accepts whichever key it
// presents, as happens only before the key has been recorded.
type Tablet struct {
	Host    string
	HostKey string
}

// Listing is a tablet's documents and the host key it presented.
type Listing struct {
	Documents []models.RemarkableDocument
	HostKey   string
}

// CoverRequest names a document's cover: the thumbnail of its page PageID.
type CoverRequest struct {
	DocumentUUID string
	PageID       string
}

// API is the front interface sync depends on to reach tablets, so it can be
// substituted with a fake in tests.
type API interface {
	// ListDocuments reads every document on the tablet.
	ListDocuments(tablet Tablet) (Listing, error)
	// Pair signs in to the tablet with its password, once, to install the
	// server's key, checks the key then works, and returns the host key the
	// tablet presented for recording.
	Pair(tablet Tablet, password string) (string, error)
	// CoverImages fetches the PNG thumbnail of each requested cover, keyed by
	// document UUID, leaving out covers the tablet hasn't rendered yet.
	CoverImages(tablet Tablet, requests []CoverRequest) (map[string][]byte, error)
}
