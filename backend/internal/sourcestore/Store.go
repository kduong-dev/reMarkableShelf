package sourcestore

import "github.com/kduong-dev/reMarkableShelf/backend/internal/models"

// Store holds the ebook sources ebooks are fetched from, in the order
// they're tried.
type Store interface {
	// List lists every source in priority order.
	List() ([]models.EbookSource, error)
	Get(id string) (models.EbookSource, error)
	// Create adds a source last in line, enabled, refusing a second one of
	// the same kind at the same URL.
	Create(source models.EbookSource) (models.EbookSource, error)
	SetEnabled(id string, enabled bool) (models.EbookSource, error)
	// Reorder puts the sources in the order of ids, which must list each
	// of them once.
	Reorder(ids []string) error
	// Delete removes a source, refusing the built-in one.
	Delete(id string) error
}
