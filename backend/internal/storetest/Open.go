package storetest

import (
	"path/filepath"
	"testing"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/database"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/documentstore"
)

// Open opens a database in the test's temporary directory, closed when the
// test ends, and returns its stores.
func Open(t *testing.T) (bookstore.Store, devicestore.Store, documentstore.Store) {
	opened, err := database.Open(filepath.Join(t.TempDir(), "books.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })
	return bookstore.NewSQLiteStore(opened), devicestore.NewSQLiteStore(opened), documentstore.NewSQLiteStore(opened)
}
