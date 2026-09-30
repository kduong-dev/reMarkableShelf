package documentstore

import (
	"time"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

// Store holds the documents synced from each tablet and the books
// they're linked to.
type Store interface {
	// Upsert replaces the known document set for a device with the
	// freshly-synced list, preserving any existing link to a book. A
	// document missing from the list is marked removed rather than deleted,
	// so it keeps its link should it reappear, as it would had one sync
	// failed to read it.
	Upsert(deviceID string, documents []models.RemarkableDocument) error
	// ListByDevice lists the documents on a device, leaving out removed ones,
	// as do the other lists.
	ListByDevice(deviceID string) ([]models.RemarkableDocument, error)
	// ListByBook lists the tablet documents linked to a book.
	ListByBook(bookID string) ([]models.RemarkableDocument, error)
	// ReplaceFolders replaces a device's folders with the freshly-synced ones.
	ReplaceFolders(deviceID string, folders []models.RemarkableFolder) error
	ListFolders(deviceID string) ([]models.RemarkableFolder, error)
	// LinkToBook links a document to a book, which also clears any mark that
	// it's not a book.
	LinkToBook(deviceID, documentUUID, bookID string) error
	// SetNotABook marks a document as not a book, unlinking it, or clears
	// the mark.
	SetNotABook(deviceID, documentUUID string, notABook bool) error
	// Unlink detaches a document from its book and stops sync linking it
	// again by title.
	Unlink(deviceID, documentUUID string) error
	// UnlinkBook is Unlink for every document linked to the book, on every
	// device, including ones since removed from their tablet, so sync
	// neither links them by title nor imports them as books again.
	UnlinkBook(bookID string) error
	// SetCover stores a document's cover as fetched when the document was
	// last modified at checkedAt. A nil image records the check but keeps
	// any cover already stored.
	SetCover(deviceID, documentUUID string, image []byte, checkedAt time.Time) error
	GetCover(deviceID, documentUUID string) ([]byte, error)
}
