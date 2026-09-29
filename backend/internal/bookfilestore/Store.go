package bookfilestore

import (
	"context"
	"io"
	"time"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

// Store keeps each book's ebook file and records which tablets it has been
// copied to. Only the methods moving the file itself take a context.
type Store interface {
	// Save detects the file's format and keeps it for the book, replacing
	// any file it had. A replaced file keeps its deliveries, so it isn't
	// copied to tablets again.
	Save(ctx context.Context, input SaveInput) (models.BookFile, error)
	Get(bookID string) (models.BookFile, error)
	// Open returns the book's file and its contents, which the caller closes.
	Open(ctx context.Context, bookID string) (models.BookFile, io.ReadCloser, error)
	// Delete removes the book's file and forgets its deliveries, leaving
	// copies on tablets alone.
	Delete(ctx context.Context, bookID string) error
	// ListUndelivered lists the files not yet copied to the device.
	ListUndelivered(deviceID string) ([]models.BookFile, error)
	RecordDelivery(input RecordDeliveryInput) error
	// ListUnloaded lists the copies on the device its reading app hasn't
	// loaded yet.
	ListUnloaded(deviceID string) ([]models.Delivery, error)
	// MarkLoaded records that the device's reading app loaded every copy on
	// it at loadedAt.
	MarkLoaded(deviceID string, loadedAt time.Time) error
}

type SaveInput struct {
	BookID string
	Source models.BookFileSource
	Body   io.Reader
}

// RecordDeliveryInput is a book's file copied to a device as the document
// DocumentUUID, with a nil LoadedAt while the reading app has yet to load it.
type RecordDeliveryInput struct {
	BookID       string
	DeviceID     string
	DocumentUUID string
	CopiedAt     time.Time
	LoadedAt     *time.Time
}
