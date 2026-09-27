package remarkable

import (
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"golang.org/x/crypto/ssh"
)

// Config is how the server signs in to tablets: as User on Port, with
// Signer, its own key, which pairing installs on each tablet.
type Config struct {
	User   string
	Port   string
	Signer ssh.Signer
}

// API is the front interface sync depends on to reach tablets, so it can be
// substituted with a fake in tests.
type API interface {
	// ListDocuments reads every document on the tablet at host.
	ListDocuments(host string) ([]models.RemarkableDocument, error)
	// Pair signs in to the tablet at host with its password, once, to
	// install the server's key, and checks the key then works.
	Pair(host, password string) error
}
