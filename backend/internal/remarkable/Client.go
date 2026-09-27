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

// CoverRequest names a document's cover: the thumbnail of its page PageID.
type CoverRequest struct {
	DocumentUUID string
	PageID       string
}

// API is the front interface sync depends on to reach tablets, so it can be
// substituted with a fake in tests.
type API interface {
	// ListDocuments reads every document on the tablet at host.
	ListDocuments(host string) ([]models.RemarkableDocument, error)
	// Pair signs in to the tablet at host with its password, once, to
	// install the server's key, and checks the key then works.
	Pair(host, password string) error
	// CoverImages fetches the PNG thumbnail of each requested cover, keyed by
	// document UUID, leaving out covers the tablet hasn't rendered yet.
	CoverImages(host string, requests []CoverRequest) (map[string][]byte, error)
}
