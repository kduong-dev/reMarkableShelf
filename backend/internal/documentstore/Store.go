package documentstore

import (
	"time"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

// Store holds the documents synced from each tablet and the books
// they're linked to.
type Store interface {
	// Upsert replaces the known document set for a device with the
	// freshly-synced list, preserving any existing link to a book.
	Upsert(deviceID string, documents []models.RemarkableDocument) error
	ListByDevice(deviceID string) ([]models.RemarkableDocument, error)
	// ListByBook lists the tablet documents linked to a book.
	ListByBook(bookID string) ([]models.RemarkableDocument, error)
	LinkToBook(deviceID, documentUUID, bookID string) error
	// Unlink detaches a document from its book and stops sync linking it
	// again by title.
	Unlink(deviceID, documentUUID string) error
	// SetCover stores a document's cover as fetched when the document was
	// last modified at checkedAt. A nil image records the check but keeps
	// any cover already stored.
	SetCover(deviceID, documentUUID string, image []byte, checkedAt time.Time) error
	GetCover(deviceID, documentUUID string) ([]byte, error)
}
