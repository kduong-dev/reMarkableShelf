package storetest

import (
	"path/filepath"
	"testing"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookfilestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/database"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/documentstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/sourcestore"
)

// Open opens a database in the test's temporary directory, closed when the
// test ends, and returns its stores.
func Open(t *testing.T) (bookstore.Store, devicestore.Store, documentstore.Store) {
	stores := OpenStores(t)
	return stores.BookStore, stores.DeviceStore, stores.DocumentStore
}

// Stores are the stores of one test database, with book files kept in
// StorageService.
type Stores struct {
	BookFileStore  bookfilestore.Store
	BookStore      bookstore.Store
	DeviceStore    devicestore.Store
	DocumentStore  documentstore.Store
	SourceStore    sourcestore.Store
	StorageService *FakeStorageService
}

// OpenStores is Open with the book file store too.
func OpenStores(t *testing.T) Stores {
	opened, err := database.Open(filepath.Join(t.TempDir(), "books.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })
	storageService := NewFakeStorageService()
	return Stores{
		BookFileStore:  bookfilestore.NewStorageServiceStore(opened, storageService),
		BookStore:      bookstore.NewSQLiteStore(opened),
		DeviceStore:    devicestore.NewSQLiteStore(opened),
		DocumentStore:  documentstore.NewSQLiteStore(opened),
		SourceStore:    sourcestore.NewSQLiteStore(opened),
		StorageService: storageService,
	}
}
