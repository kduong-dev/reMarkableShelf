package bookfilestore

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/kduong-dev/goutil/logx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

var contentTypes = map[models.FileType]string{
	models.FileTypeEPUB: "application/epub+zip",
	models.FileTypePDF:  "application/pdf",
}

// StorageServiceStore keeps files in the storage service and records them,
// by their storage file id, in the database.
type StorageServiceStore struct {
	database       *sql.DB
	storageService storageservice.Client
}

func NewStorageServiceStore(database *sql.DB, storageService storageservice.Client) *StorageServiceStore {
	return &StorageServiceStore{database: database, storageService: storageService}
}

func (store *StorageServiceStore) Save(ctx context.Context, input SaveInput) (models.BookFile, error) {
	reader := bufio.NewReader(input.Body)
	header, err := reader.Peek(formatHeaderLength)
	if err != nil && !errors.Is(err, io.EOF) {
		return models.BookFile{}, fmt.Errorf("reading book file: %w", err)
	}
	format, err := DetectFormat(header)
	if err != nil {
		return models.BookFile{}, err
	}
	previousID, err := store.storageFileID(input.BookID)
	if err != nil && !errors.Is(err, ErrFileNotFound) {
		return models.BookFile{}, err
	}
	// The storage service adds a file per upload rather than replacing one
	// under the same key, so the previous file is deleted once this one is
	// recorded.
	uploaded, err := storageservice.UploadFile(ctx, store.storageService, storageservice.UploadFileInput{
		Key:         fmt.Sprintf("books/%s.%s", input.BookID, format),
		ContentType: contentTypes[format],
		Body:        reader,
	})
	if err != nil {
		return models.BookFile{}, fmt.Errorf("%w: uploading book file: %w", ErrStorageUnavailable, err)
	}
	_, err = store.database.Exec(`
		INSERT INTO book_files (book_id, storage_file_id, format, size, source, source_name, saved_at) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (book_id) DO UPDATE SET storage_file_id = excluded.storage_file_id, format = excluded.format,
			size = excluded.size, source = excluded.source, source_name = excluded.source_name, saved_at = excluded.saved_at`,
		input.BookID, uploaded.ID, format, uploaded.Size, input.Source, input.SourceName, time.Now().UTC())
	if err != nil {
		store.deleteStorageFile(ctx, uploaded.ID)
		return models.BookFile{}, fmt.Errorf("recording book file: %w", err)
	}
	if previousID != "" {
		store.deleteStorageFile(ctx, previousID)
	}
	return store.Get(input.BookID)
}

func (store *StorageServiceStore) Get(bookID string) (models.BookFile, error) {
	var file models.BookFile
	err := store.database.QueryRow(`SELECT book_id, format, size, source, source_name, saved_at FROM book_files WHERE book_id = ?`, bookID).
		Scan(&file.BookID, &file.Format, &file.Size, &file.Source, &file.SourceName, &file.SavedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.BookFile{}, fmt.Errorf("book %q: %w", bookID, ErrFileNotFound)
	}
	if err != nil {
		return models.BookFile{}, fmt.Errorf("getting book file: %w", err)
	}
	file.Deliveries, err = store.listDeliveries(`WHERE book_id = ?`, bookID)
	return file, err
}

func (store *StorageServiceStore) Open(ctx context.Context, bookID string) (models.BookFile, io.ReadCloser, error) {
	file, err := store.Get(bookID)
	if err != nil {
		return models.BookFile{}, nil, err
	}
	storageFileID, err := store.storageFileID(bookID)
	if err != nil {
		return models.BookFile{}, nil, err
	}
	download, err := store.storageService.DownloadFile(ctx, storageservice.DownloadFileInput{FileID: storageFileID})
	if err != nil {
		return models.BookFile{}, nil, fmt.Errorf("%w: downloading book file: %w", ErrStorageUnavailable, err)
	}
	return file, download.Body, nil
}

func (store *StorageServiceStore) Delete(ctx context.Context, bookID string) error {
	storageFileID, err := store.storageFileID(bookID)
	if err != nil {
		return err
	}
	if _, err := store.database.Exec(`DELETE FROM book_files WHERE book_id = ?`, bookID); err != nil {
		return fmt.Errorf("deleting book file: %w", err)
	}
	store.deleteStorageFile(ctx, storageFileID)
	return nil
}

func (store *StorageServiceStore) ListUndelivered(deviceID string) ([]models.BookFile, error) {
	rows, err := store.database.Query(`
		SELECT book_id, format, size, source, source_name, saved_at FROM book_files
		 WHERE book_id NOT IN (SELECT book_id FROM book_file_deliveries WHERE device_id = ?)
		 ORDER BY saved_at`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("listing undelivered book files: %w", err)
	}
	defer rows.Close()
	files := []models.BookFile{}
	for rows.Next() {
		var file models.BookFile
		if err := rows.Scan(&file.BookID, &file.Format, &file.Size, &file.Source, &file.SourceName, &file.SavedAt); err != nil {
			return nil, fmt.Errorf("listing undelivered book files: %w", err)
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

func (store *StorageServiceStore) RecordDelivery(input RecordDeliveryInput) error {
	_, err := store.database.Exec(`
		INSERT INTO book_file_deliveries (book_id, device_id, document_uuid, copied_at, loaded_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (book_id, device_id) DO UPDATE SET document_uuid = excluded.document_uuid, copied_at = excluded.copied_at, loaded_at = excluded.loaded_at`,
		input.BookID, input.DeviceID, input.DocumentUUID, input.CopiedAt, input.LoadedAt)
	if err != nil {
		return fmt.Errorf("recording delivery: %w", err)
	}
	return nil
}

func (store *StorageServiceStore) ListUnloaded(deviceID string) ([]models.Delivery, error) {
	return store.listDeliveries(`WHERE device_id = ? AND loaded_at IS NULL`, deviceID)
}

func (store *StorageServiceStore) MarkLoaded(deviceID string, loadedAt time.Time) error {
	_, err := store.database.Exec(`UPDATE book_file_deliveries SET loaded_at = ? WHERE device_id = ? AND loaded_at IS NULL`, loadedAt, deviceID)
	if err != nil {
		return fmt.Errorf("marking deliveries loaded: %w", err)
	}
	return nil
}

func (store *StorageServiceStore) storageFileID(bookID string) (string, error) {
	var storageFileID string
	err := store.database.QueryRow(`SELECT storage_file_id FROM book_files WHERE book_id = ?`, bookID).Scan(&storageFileID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("book %q: %w", bookID, ErrFileNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("getting book file: %w", err)
	}
	return storageFileID, nil
}

// deleteStorageFile deletes a file no longer recorded for any book. Failing
// to only leaves an unused file in the storage service, so it's logged.
func (store *StorageServiceStore) deleteStorageFile(ctx context.Context, storageFileID string) {
	err := store.storageService.DeleteFile(ctx, storageservice.DeleteFileInput{FileID: storageFileID})
	if err != nil && !errors.Is(err, storageservice.ErrFileNotFound) {
		logx.Warnf("deleting storage file %s: %v", storageFileID, err)
	}
}

func (store *StorageServiceStore) listDeliveries(where string, argument string) ([]models.Delivery, error) {
	rows, err := store.database.Query(`SELECT device_id, document_uuid, copied_at, loaded_at FROM book_file_deliveries `+where+` ORDER BY copied_at`, argument)
	if err != nil {
		return nil, fmt.Errorf("listing deliveries: %w", err)
	}
	defer rows.Close()
	deliveries := []models.Delivery{}
	for rows.Next() {
		var delivery models.Delivery
		if err := rows.Scan(&delivery.DeviceID, &delivery.DocumentUUID, &delivery.CopiedAt, &delivery.LoadedAt); err != nil {
			return nil, fmt.Errorf("listing deliveries: %w", err)
		}
		deliveries = append(deliveries, delivery)
	}
	return deliveries, rows.Err()
}
