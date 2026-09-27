package devicesync

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/kduong-dev/goutil/logx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/remarkable"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/store"
)

// Syncer copies a tablet's documents and reading positions into the store.
// Syncs run one at a time, so a manual sync and a background one can't
// interleave their writes.
type Syncer struct {
	store      *store.Store
	remarkable remarkable.API
	mutex      sync.Mutex
}

func NewSyncer(store *store.Store, remarkable remarkable.API) *Syncer {
	return &Syncer{store: store, remarkable: remarkable}
}

// SyncDevice reads every document on the device, upserts them, auto-links
// pdf/epub documents to an existing book by exact (case-insensitive) title
// match, then moves linked books' bookmarks to newer tablet positions.
// Notebooks are stored but never auto-linked — see models.FileType.
func (syncer *Syncer) SyncDevice(deviceID string) ([]models.RemarkableDocument, error) {
	syncer.mutex.Lock()
	defer syncer.mutex.Unlock()
	device, err := syncer.store.GetDevice(deviceID)
	if err != nil {
		return nil, err
	}
	documents, err := syncer.remarkable.ListDocuments(device.Host)
	if err != nil {
		return nil, err
	}
	if err := syncer.store.UpsertDocuments(device.ID, documents); err != nil {
		return nil, err
	}
	if err := syncer.store.TouchDeviceSyncedAt(device.ID, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := syncer.autoLinkDocuments(device.ID); err != nil {
		return nil, err
	}
	syncedDocuments, err := syncer.store.ListDocumentsByDevice(device.ID)
	if err != nil {
		return nil, err
	}
	if err := syncer.applyTabletProgress(syncedDocuments); err != nil {
		return nil, err
	}
	return syncedDocuments, nil
}

// Run syncs every registered device each interval until ctx is done. A
// tablet that's asleep or off the network fails to connect, which is
// expected, so failures are logged and retried on the next tick.
func (syncer *Syncer) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			syncer.syncAllDevices()
		}
	}
}

func (syncer *Syncer) syncAllDevices() {
	devices, err := syncer.store.ListDevices()
	if err != nil {
		logx.Warnf("background sync: listing devices: %v", err)
		return
	}
	for _, device := range devices {
		if _, err := syncer.SyncDevice(device.ID); err != nil {
			logx.Warnf("background sync of %s (%s): %v", device.Name, device.Host, err)
		}
	}
}

func (syncer *Syncer) autoLinkDocuments(deviceID string) error {
	documents, err := syncer.store.ListDocumentsByDevice(deviceID)
	if err != nil {
		return err
	}
	books, err := syncer.store.ListBooks()
	if err != nil {
		return err
	}
	booksByTitle := make(map[string]models.Book, len(books))
	for _, book := range books {
		booksByTitle[strings.ToLower(strings.TrimSpace(book.Title))] = book
	}
	for _, document := range documents {
		if document.LinkedBookID != nil || document.FileType == models.FileTypeNotebook {
			continue
		}
		book, found := booksByTitle[strings.ToLower(strings.TrimSpace(document.Title))]
		if !found {
			continue
		}
		if err := syncer.store.LinkDocumentToBook(deviceID, document.UUID, book.ID); err != nil {
			return err
		}
	}
	return nil
}

func (syncer *Syncer) applyTabletProgress(documents []models.RemarkableDocument) error {
	for _, document := range documents {
		if document.LinkedBookID == nil {
			continue
		}
		book, err := syncer.store.GetBook(*document.LinkedBookID)
		if err != nil {
			return err
		}
		input, ok := tabletProgress(book, document)
		if !ok {
			continue
		}
		if _, err := syncer.store.SetTabletProgress(input); err != nil {
			return err
		}
	}
	return nil
}
