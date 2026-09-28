package store

import (
	"database/sql"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/database"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/documentstore"
)

// Store is the opened database, with a store per table it holds.
type Store struct {
	Books     bookstore.Store
	Devices   devicestore.Store
	Documents documentstore.Store
	database  *sql.DB
}

func Open(path string) (*Store, error) {
	opened, err := database.Open(path)
	if err != nil {
		return nil, err
	}
	return &Store{
		Books:     bookstore.NewSQLiteStore(opened),
		Devices:   devicestore.NewSQLiteStore(opened),
		Documents: documentstore.NewSQLiteStore(opened),
		database:  opened,
	}, nil
}

func (store *Store) Close() error {
	return store.database.Close()
}
