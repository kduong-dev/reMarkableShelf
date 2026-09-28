package devicesync

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/ansel1/merry"
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
	device, err := syncer.store.Devices.Get(deviceID)
	if err != nil {
		return nil, err
	}
	listing, err := syncer.remarkable.ListDocuments(tabletOf(device))
	switch {
	case merry.Is(err, remarkable.ErrHostKeyChanged):
		if markErr := syncer.store.Devices.MarkIdentityChanged(device.ID); markErr != nil {
			return nil, markErr
		}
	case merry.Is(err, remarkable.ErrNotPaired) && device.PairedAt != nil:
		if unpairErr := syncer.store.Devices.SetPairedAt(device.ID, nil); unpairErr != nil {
			return nil, unpairErr
		}
	}
	if err != nil {
		return nil, err
	}
	if device.HostKey == "" {
		if err := syncer.store.Devices.SetHostKey(device.ID, listing.HostKey); err != nil {
			return nil, err
		}
		device.HostKey = listing.HostKey
	}
	documents := listing.Documents
	if err := syncer.store.Documents.Upsert(device.ID, documents); err != nil {
		return nil, err
	}
	if err := syncer.store.Devices.TouchSyncedAt(device.ID, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := syncer.autoLinkDocuments(device.ID); err != nil {
		return nil, err
	}
	if err := syncer.fetchCovers(device, documents); err != nil {
		return nil, err
	}
	syncedDocuments, err := syncer.store.Documents.ListByDevice(device.ID)
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
	devices, err := syncer.store.Devices.List()
	if err != nil {
		logx.Warnf("background sync: listing devices: %v", err)
		return
	}
	for _, device := range devices {
		// An unpaired tablet can't be signed in to until it's paired from
		// the Sync page, so retrying it would only log the same failure.
		if device.PairedAt == nil {
			continue
		}
		if _, err := syncer.SyncDevice(device.ID); err != nil {
			logx.Warnf("background sync of %s (%s): %v", device.Name, device.Host, err)
		}
	}
}

// RegisterDeviceInput is a tablet to add, with the password used once to
// pair it; the password is never stored.
type RegisterDeviceInput struct {
	Name     string
	Host     string
	Password string
}

// RegisterDevice pairs the tablet and, only once that works, adds it, so a
// wrong password or address doesn't leave a device that can't sync.
func (syncer *Syncer) RegisterDevice(input RegisterDeviceInput) (models.Device, error) {
	input.Name, input.Host = strings.TrimSpace(input.Name), strings.TrimSpace(input.Host)
	if input.Name == "" || input.Host == "" {
		return models.Device{}, merry.Here(ErrDeviceDetailsRequired)
	}
	if input.Password == "" {
		return models.Device{}, merry.Here(ErrPasswordRequired)
	}
	devices, err := syncer.store.Devices.List()
	if err != nil {
		return models.Device{}, err
	}
	for _, device := range devices {
		if strings.EqualFold(device.Host, input.Host) {
			return models.Device{}, merry.Here(ErrDeviceAlreadyRegistered).
				WithUserMessagef("%s is already registered as %q; pair that one again instead of adding it twice", device.Host, device.Name)
		}
	}
	hostKey, err := syncer.remarkable.Pair(remarkable.Tablet{Host: input.Host}, input.Password)
	if err != nil {
		return models.Device{}, err
	}
	pairedAt := time.Now().UTC()
	return syncer.store.Devices.Create(models.Device{Name: input.Name, Host: input.Host, PairedAt: &pairedAt, HostKey: hostKey})
}

// PairDeviceInput pairs a registered device again. The tablet must present
// its recorded host key unless AcceptNewIdentity confirms it was
// factory-reset or replaced; otherwise an impostor could trigger a failed
// sync to collect the password.
type PairDeviceInput struct {
	DeviceID          string
	Password          string
	AcceptNewIdentity bool
}

func (syncer *Syncer) PairDevice(input PairDeviceInput) (models.Device, error) {
	if input.Password == "" {
		return models.Device{}, merry.Here(ErrPasswordRequired)
	}
	device, err := syncer.store.Devices.Get(input.DeviceID)
	if err != nil {
		return models.Device{}, err
	}
	tablet := tabletOf(device)
	if input.AcceptNewIdentity {
		tablet.HostKey = ""
	}
	hostKey, err := syncer.remarkable.Pair(tablet, input.Password)
	if merry.Is(err, remarkable.ErrHostKeyChanged) {
		if markErr := syncer.store.Devices.MarkIdentityChanged(device.ID); markErr != nil {
			return models.Device{}, markErr
		}
	}
	if err != nil {
		return models.Device{}, err
	}
	pairedAt := time.Now().UTC()
	if err := syncer.store.Devices.MarkPaired(device.ID, pairedAt, hostKey); err != nil {
		return models.Device{}, err
	}
	return syncer.store.Devices.Get(device.ID)
}

func tabletOf(device models.Device) remarkable.Tablet {
	return remarkable.Tablet{Host: device.Host, HostKey: device.HostKey}
}

// LinkDocument links a tablet document to a book and applies the position
// already synced for it, so the book shows progress without waiting for the
// next sync.
func (syncer *Syncer) LinkDocument(deviceID, documentUUID, bookID string) error {
	syncer.mutex.Lock()
	defer syncer.mutex.Unlock()
	if err := syncer.store.Documents.LinkToBook(deviceID, documentUUID, bookID); err != nil {
		return err
	}
	documents, err := syncer.store.Documents.ListByDevice(deviceID)
	if err != nil {
		return err
	}
	for _, document := range documents {
		if document.UUID == documentUUID {
			return syncer.applyTabletProgress([]models.RemarkableDocument{document})
		}
	}
	return nil
}

// fetchCovers copies the cover thumbnail of each pdf/epub document that
// has none yet or has changed since its cover was fetched. The tablet
// renders thumbnails lazily, so a missing one is retried on later syncs.
// Covers are a nicety, so failing to fetch them is logged, not returned.
func (syncer *Syncer) fetchCovers(device models.Device, listed []models.RemarkableDocument) error {
	stored, err := syncer.store.Documents.ListByDevice(device.ID)
	if err != nil {
		return err
	}
	storedByUUID := make(map[string]models.RemarkableDocument, len(stored))
	for _, document := range stored {
		storedByUUID[document.UUID] = document
	}
	var requests []remarkable.CoverRequest
	modifiedAt := make(map[string]time.Time)
	for _, document := range listed {
		if document.FileType == models.FileTypeNotebook || document.CoverPageID == "" {
			continue
		}
		current := storedByUUID[document.UUID]
		if current.HasCover && current.CoverCheckedAt != nil && !document.LastModified.After(*current.CoverCheckedAt) {
			continue
		}
		requests = append(requests, remarkable.CoverRequest{DocumentUUID: document.UUID, PageID: document.CoverPageID})
		modifiedAt[document.UUID] = document.LastModified
	}
	images, err := syncer.remarkable.CoverImages(tabletOf(device), requests)
	if err != nil {
		logx.Warnf("fetching covers from %s (%s): %v", device.Name, device.Host, err)
		return nil
	}
	for _, request := range requests {
		if err := syncer.store.Documents.SetCover(device.ID, request.DocumentUUID, images[request.DocumentUUID], modifiedAt[request.DocumentUUID]); err != nil {
			return err
		}
	}
	return nil
}

func (syncer *Syncer) autoLinkDocuments(deviceID string) error {
	documents, err := syncer.store.Documents.ListByDevice(deviceID)
	if err != nil {
		return err
	}
	books, err := syncer.store.Books.List()
	if err != nil {
		return err
	}
	booksByTitle := make(map[string]models.Book, len(books))
	for _, book := range books {
		booksByTitle[strings.ToLower(strings.TrimSpace(book.Title))] = book
	}
	for _, document := range documents {
		if document.LinkedBookID != nil || document.AutoLinkDismissed || document.FileType == models.FileTypeNotebook {
			continue
		}
		book, found := booksByTitle[strings.ToLower(strings.TrimSpace(document.Title))]
		if !found {
			continue
		}
		if err := syncer.store.Documents.LinkToBook(deviceID, document.UUID, book.ID); err != nil {
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
		book, err := syncer.store.Books.Get(*document.LinkedBookID)
		if err != nil {
			return err
		}
		if input, ok := tabletProgress(book, document); ok {
			if _, err := syncer.store.Books.SetTabletProgress(input); err != nil {
				return err
			}
			continue
		}
		// With no newer position to take, still align the book to the
		// tablet's page count; the store keeps the bookmark's place.
		if document.PageCount != nil && (book.PageCount == nil || *book.PageCount != *document.PageCount) {
			if _, err := syncer.store.Books.Update(book.ID, models.Book{PageCount: document.PageCount}); err != nil {
				return err
			}
		}
	}
	return nil
}

// AlignBook brings a book back in line with its linked tablet documents
// after it's edited, e.g. when matching it to an edition set another page
// count.
func (syncer *Syncer) AlignBook(bookID string) error {
	syncer.mutex.Lock()
	defer syncer.mutex.Unlock()
	documents, err := syncer.store.Documents.ListByBook(bookID)
	if err != nil {
		return err
	}
	return syncer.applyTabletProgress(documents)
}
