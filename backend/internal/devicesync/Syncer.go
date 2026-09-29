package devicesync

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kduong-dev/goutil/logx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookfilestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/documentstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/remarkable"
)

// Syncer copies a tablet's documents and reading positions into the store.
// Syncs run one at a time, so a manual sync and a background one can't
// interleave their writes.
type Syncer struct {
	bookFileStore bookfilestore.Store
	bookStore     bookstore.Store
	deviceStore   devicestore.Store
	documentStore documentstore.Store
	remarkable    remarkable.API
	mutex         sync.Mutex
}

type NewSyncerInput struct {
	BookFileStore bookfilestore.Store
	BookStore     bookstore.Store
	DeviceStore   devicestore.Store
	DocumentStore documentstore.Store
	Remarkable    remarkable.API
}

func NewSyncer(input NewSyncerInput) *Syncer {
	return &Syncer{
		bookFileStore: input.BookFileStore,
		bookStore:     input.BookStore,
		deviceStore:   input.DeviceStore,
		documentStore: input.DocumentStore,
		remarkable:    input.Remarkable,
	}
}

// SyncDevice reads every document and folder on the device, upserts them, auto-links
// pdf/epub documents to an existing book by exact (case-insensitive) title
// match, copies book files to it for books with no document there, then
// moves linked books' bookmarks to newer tablet positions. Notebooks are stored but never auto-linked — see
// models.FileType. Copied books show on the tablet once its reading app
// restarts, which SyncDeviceNow does.
func (syncer *Syncer) SyncDevice(deviceID string) ([]models.RemarkableDocument, error) {
	return syncer.syncDevice(deviceID, false)
}

// SyncDeviceNow is SyncDevice for a sync the user asked for, which also
// restarts the tablet's reading app when copied books are waiting to show,
// closing whatever is open. Background syncs never restart it, so they
// can't interrupt reading.
func (syncer *Syncer) SyncDeviceNow(deviceID string) ([]models.RemarkableDocument, error) {
	return syncer.syncDevice(deviceID, true)
}

func (syncer *Syncer) syncDevice(deviceID string, restartApp bool) ([]models.RemarkableDocument, error) {
	syncer.mutex.Lock()
	defer syncer.mutex.Unlock()
	device, err := syncer.deviceStore.Get(deviceID)
	if err != nil {
		return nil, err
	}
	listing, err := syncer.remarkable.ListDocuments(tabletOf(device))
	switch {
	case errors.Is(err, remarkable.ErrHostKeyChanged):
		if markErr := syncer.deviceStore.MarkIdentityChanged(device.ID); markErr != nil {
			return nil, markErr
		}
	case errors.Is(err, remarkable.ErrNotPaired) && device.PairedAt != nil:
		if unpairErr := syncer.deviceStore.SetPairedAt(device.ID, nil); unpairErr != nil {
			return nil, unpairErr
		}
	}
	if err != nil {
		return nil, err
	}
	if device.HostKey == "" {
		if err := syncer.deviceStore.SetHostKey(device.ID, listing.HostKey); err != nil {
			return nil, err
		}
		device.HostKey = listing.HostKey
	}
	documents := listing.Documents
	if err := syncer.documentStore.Upsert(device.ID, documents); err != nil {
		return nil, err
	}
	if err := syncer.documentStore.ReplaceFolders(device.ID, listing.Folders); err != nil {
		return nil, err
	}
	if err := syncer.deviceStore.TouchSyncedAt(device.ID, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := syncer.autoLinkDocuments(device.ID); err != nil {
		return nil, err
	}
	copied, createdFolder, err := syncer.copyBooks(device, listing.Folders)
	if err != nil {
		return nil, err
	}
	if createdFolder != nil {
		if err := syncer.documentStore.ReplaceFolders(device.ID, append(slices.Clone(listing.Folders), *createdFolder)); err != nil {
			return nil, err
		}
	}
	if len(copied) > 0 {
		if documents, err = syncer.addCopiedDocuments(device.ID, documents, copied); err != nil {
			return nil, err
		}
	}
	if err := syncer.fetchCovers(device, documents); err != nil {
		return nil, err
	}
	syncedDocuments, err := syncer.documentStore.ListByDevice(device.ID)
	if err != nil {
		return nil, err
	}
	if err := syncer.applyTabletProgress(syncedDocuments); err != nil {
		return nil, err
	}
	if restartApp {
		if err := syncer.loadCopiedBooks(device); err != nil {
			return nil, err
		}
	}
	return syncedDocuments, nil
}

// copiedBook is a book's file copied to a tablet as a new document.
type copiedBook struct {
	book     models.Book
	document models.RemarkableDocument
}

// DownloadsFolderTitle names the tablet folder books saved on the server are
// copied into.
const DownloadsFolderTitle = "Downloads"

// copyBooks copies each book file not yet delivered to the device into its
// downloads folder, returning the folder too if it had to be created. A
// book already linked to a document on the device is recorded as delivered
// instead of being copied again. A book that fails to copy is logged and
// retried on the next sync; the books after it are still copied unless the
// tablet has gone out of reach.
func (syncer *Syncer) copyBooks(device models.Device, folders []models.RemarkableFolder) ([]copiedBook, *models.RemarkableFolder, error) {
	var copied []copiedBook
	var createdFolder *models.RemarkableFolder
	downloadsFolderUUID := ""
	files, err := syncer.bookFileStore.ListUndelivered(device.ID)
	if err != nil {
		return nil, nil, err
	}
	for _, file := range files {
		linked, err := syncer.linkedDocument(device.ID, file.BookID)
		if err != nil {
			return nil, nil, err
		}
		now := time.Now().UTC()
		if linked != nil {
			err := syncer.bookFileStore.RecordDelivery(bookfilestore.RecordDeliveryInput{
				BookID: file.BookID, DeviceID: device.ID, DocumentUUID: linked.UUID, CopiedAt: now, LoadedAt: &now,
			})
			if err != nil {
				return nil, nil, err
			}
			continue
		}
		book, err := syncer.bookStore.Get(file.BookID)
		if err != nil {
			return nil, nil, err
		}
		if downloadsFolderUUID == "" {
			downloadsFolderUUID, createdFolder, err = syncer.downloadsFolder(device, folders)
			if errors.Is(err, errTabletFailed) {
				logx.Warnf("creating the %s folder on %s (%s): %v", DownloadsFolderTitle, device.Name, device.Host, err)
				return copied, nil, nil
			}
			if err != nil {
				return nil, nil, err
			}
		}
		documentUUID := uuid.NewString()
		if err := syncer.copyBook(device, book, file, remarkable.CopyDocumentInput{UUID: documentUUID, ParentUUID: downloadsFolderUUID}); err != nil {
			logx.Warnf("copying %q to %s (%s): %v", book.Title, device.Name, device.Host, err)
			if tabletOutOfReach(err) {
				return copied, createdFolder, nil
			}
			continue
		}
		err = syncer.bookFileStore.RecordDelivery(bookfilestore.RecordDeliveryInput{
			BookID: file.BookID, DeviceID: device.ID, DocumentUUID: documentUUID, CopiedAt: now,
		})
		if err != nil {
			return nil, nil, err
		}
		copied = append(copied, copiedBook{book: book, document: models.RemarkableDocument{
			UUID:         documentUUID,
			DeviceID:     device.ID,
			Title:        book.Title,
			FileType:     file.Format,
			LastModified: now,
			ParentUUID:   downloadsFolderUUID,
		}})
	}
	return copied, createdFolder, nil
}

// tabletOutOfReach reports whether err means no more can be copied to the
// tablet this sync.
func tabletOutOfReach(err error) bool {
	return errors.Is(err, remarkable.ErrTabletUnreachable) ||
		errors.Is(err, remarkable.ErrNotPaired) ||
		errors.Is(err, remarkable.ErrHostKeyChanged)
}

// errTabletFailed marks a failure to write to the tablet, which ends the
// copying rather than the sync.
var errTabletFailed = errors.New("writing to the tablet failed")

// downloadsFolder returns the UUID of the device's downloads folder. It's
// the folder recorded for the device while that's still on the tablet and
// out of its trash, wherever it's been moved or whatever it's been renamed
// to; otherwise a top-level folder already named Downloads; otherwise a new
// one, which is also returned.
func (syncer *Syncer) downloadsFolder(device models.Device, folders []models.RemarkableFolder) (string, *models.RemarkableFolder, error) {
	parents := make(map[string]string, len(folders))
	for _, folder := range folders {
		parents[folder.UUID] = folder.ParentUUID
	}
	if _, onTablet := parents[device.DownloadsFolderUUID]; onTablet && !inTrash(parents, device.DownloadsFolderUUID) {
		return device.DownloadsFolderUUID, nil, nil
	}
	for _, folder := range folders {
		if folder.ParentUUID == "" && strings.EqualFold(folder.Title, DownloadsFolderTitle) {
			return folder.UUID, nil, syncer.deviceStore.SetDownloadsFolder(device.ID, folder.UUID)
		}
	}
	created := models.RemarkableFolder{UUID: uuid.NewString(), Title: DownloadsFolderTitle}
	if err := syncer.remarkable.CreateFolder(tabletOf(device), created); err != nil {
		return "", nil, fmt.Errorf("%w: %w", errTabletFailed, err)
	}
	return created.UUID, &created, syncer.deviceStore.SetDownloadsFolder(device.ID, created.UUID)
}

// inTrash reports whether the folder, or any folder holding it, is in the
// trash, given each folder's parent.
func inTrash(parents map[string]string, folderUUID string) bool {
	for seen := 0; folderUUID != "" && seen <= len(parents); seen++ {
		if folderUUID == models.TrashFolderUUID {
			return true
		}
		folderUUID = parents[folderUUID]
	}
	return false
}

// copyBook copies the book's file to the device as described by input,
// which it completes with the book's title and file. The file is fetched
// in full before copying starts, since sending it over the tablet's WiFi
// can outlast the storage service's request timeout.
func (syncer *Syncer) copyBook(device models.Device, book models.Book, file models.BookFile, input remarkable.CopyDocumentInput) error {
	_, body, err := syncer.bookFileStore.Open(context.Background(), file.BookID)
	if err != nil {
		return err
	}
	defer body.Close()
	fetched, err := os.CreateTemp("", "book-*")
	if err != nil {
		return fmt.Errorf("fetching book file: %w", err)
	}
	defer os.Remove(fetched.Name())
	defer fetched.Close()
	if _, err := io.Copy(fetched, body); err != nil {
		return fmt.Errorf("fetching book file: %w", err)
	}
	if _, err := fetched.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("fetching book file: %w", err)
	}
	input.Title = book.Title
	input.FileType = file.Format
	input.Body = fetched
	return syncer.remarkable.CopyDocument(tabletOf(device), input)
}

// addCopiedDocuments adds the documents just copied to the device's
// synced documents, linked to their books, as the next listing would.
func (syncer *Syncer) addCopiedDocuments(deviceID string, listed []models.RemarkableDocument, copied []copiedBook) ([]models.RemarkableDocument, error) {
	documents := slices.Clone(listed)
	for _, delivered := range copied {
		documents = append(documents, delivered.document)
	}
	if err := syncer.documentStore.Upsert(deviceID, documents); err != nil {
		return nil, err
	}
	for _, delivered := range copied {
		if err := syncer.documentStore.LinkToBook(deviceID, delivered.document.UUID, delivered.book.ID); err != nil {
			return nil, err
		}
	}
	return documents, nil
}

func (syncer *Syncer) linkedDocument(deviceID, bookID string) (*models.RemarkableDocument, error) {
	documents, err := syncer.documentStore.ListByBook(bookID)
	if err != nil {
		return nil, err
	}
	for _, document := range documents {
		if document.DeviceID == deviceID {
			return &document, nil
		}
	}
	return nil, nil
}

// loadCopiedBooks restarts the device's reading app if books were copied
// to it since it last did, so they show in its library.
func (syncer *Syncer) loadCopiedBooks(device models.Device) error {
	unloaded, err := syncer.bookFileStore.ListUnloaded(device.ID)
	if err != nil || len(unloaded) == 0 {
		return err
	}
	if err := syncer.remarkable.RestartApp(tabletOf(device)); err != nil {
		return err
	}
	return syncer.bookFileStore.MarkLoaded(device.ID, time.Now().UTC())
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
	devices, err := syncer.deviceStore.List()
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
		return models.Device{}, ErrDeviceDetailsRequired
	}
	if input.Password == "" {
		return models.Device{}, ErrPasswordRequired
	}
	devices, err := syncer.deviceStore.List()
	if err != nil {
		return models.Device{}, err
	}
	for _, device := range devices {
		if strings.EqualFold(device.Host, input.Host) {
			return models.Device{}, DeviceAlreadyRegisteredError{Host: device.Host, Name: device.Name}
		}
	}
	hostKey, err := syncer.remarkable.Pair(remarkable.Tablet{Host: input.Host}, input.Password)
	if err != nil {
		return models.Device{}, err
	}
	pairedAt := time.Now().UTC()
	return syncer.deviceStore.Create(models.Device{Name: input.Name, Host: input.Host, PairedAt: &pairedAt, HostKey: hostKey})
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
		return models.Device{}, ErrPasswordRequired
	}
	device, err := syncer.deviceStore.Get(input.DeviceID)
	if err != nil {
		return models.Device{}, err
	}
	tablet := tabletOf(device)
	if input.AcceptNewIdentity {
		tablet.HostKey = ""
	}
	hostKey, err := syncer.remarkable.Pair(tablet, input.Password)
	if errors.Is(err, remarkable.ErrHostKeyChanged) {
		if markErr := syncer.deviceStore.MarkIdentityChanged(device.ID); markErr != nil {
			return models.Device{}, markErr
		}
	}
	if err != nil {
		return models.Device{}, err
	}
	pairedAt := time.Now().UTC()
	if err := syncer.deviceStore.MarkPaired(device.ID, pairedAt, hostKey); err != nil {
		return models.Device{}, err
	}
	return syncer.deviceStore.Get(device.ID)
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
	if err := syncer.documentStore.LinkToBook(deviceID, documentUUID, bookID); err != nil {
		return err
	}
	documents, err := syncer.documentStore.ListByDevice(deviceID)
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
	stored, err := syncer.documentStore.ListByDevice(device.ID)
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
		if err := syncer.documentStore.SetCover(device.ID, request.DocumentUUID, images[request.DocumentUUID], modifiedAt[request.DocumentUUID]); err != nil {
			return err
		}
	}
	return nil
}

func (syncer *Syncer) autoLinkDocuments(deviceID string) error {
	documents, err := syncer.documentStore.ListByDevice(deviceID)
	if err != nil {
		return err
	}
	books, err := syncer.bookStore.List()
	if err != nil {
		return err
	}
	booksByTitle := make(map[string]models.Book, len(books))
	for _, book := range books {
		booksByTitle[strings.ToLower(strings.TrimSpace(book.Title))] = book
	}
	for _, document := range documents {
		if document.LinkedBookID != nil || document.AutoLinkDismissed || document.NotABook || document.FileType == models.FileTypeNotebook {
			continue
		}
		book, found := booksByTitle[strings.ToLower(strings.TrimSpace(document.Title))]
		if !found {
			continue
		}
		if err := syncer.documentStore.LinkToBook(deviceID, document.UUID, book.ID); err != nil {
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
		book, err := syncer.bookStore.Get(*document.LinkedBookID)
		if err != nil {
			return err
		}
		if input, ok := tabletProgress(book, document); ok {
			if _, err := syncer.bookStore.SetTabletProgress(input); err != nil {
				return err
			}
			continue
		}
		// With no newer position to take, still align the book to the
		// tablet's page count; the store keeps the bookmark's place.
		if document.PageCount != nil && (book.PageCount == nil || *book.PageCount != *document.PageCount) {
			if _, err := syncer.bookStore.Update(book.ID, models.Book{PageCount: document.PageCount}); err != nil {
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
	documents, err := syncer.documentStore.ListByBook(bookID)
	if err != nil {
		return err
	}
	return syncer.applyTabletProgress(documents)
}
