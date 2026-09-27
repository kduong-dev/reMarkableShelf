package remarkable

import "github.com/kduong-dev/reMarkableShelf/backend/internal/models"

// Config holds the SSH credentials used to reach a tablet. In practice a
// user has one (or a couple) of personal tablets, so — per the setup notes
// in the repo README — these are provided once via env vars rather than
// stored per-Device in the database.
type Config struct {
	User     string
	Password string
	Port     string
}

// API is the front interface sync depends on to read a tablet's documents,
// so it can be substituted with a fake in tests.
type API interface {
	ListDocuments(host string) ([]models.RemarkableDocument, error)
}

// SSHClient reads documents straight from a tablet's filesystem over SSH.
type SSHClient struct {
	config Config
}

func NewSSHClient(config Config) *SSHClient {
	return &SSHClient{config: config}
}
