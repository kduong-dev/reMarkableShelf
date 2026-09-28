package store

import (
	"database/sql"

	"github.com/ansel1/merry"
	_ "modernc.org/sqlite"
)

// Store is the opened sqlite database, with a store per table it holds.
type Store struct {
	Books     BookStore
	Devices   DeviceStore
	Documents DocumentStore
	database  *sql.DB
}

func Open(path string) (*Store, error) {
	// SQLite ignores the schema's REFERENCES clauses, including ON DELETE
	// CASCADE and SET NULL, unless foreign keys are switched on per
	// connection, which the DSN does for every connection the pool opens.
	database, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, merry.Wrap(err)
	}
	if err := database.Ping(); err != nil {
		return nil, merry.Wrap(err)
	}
	// xochitl syncs can upsert many rows at once; sqlite only allows one
	// writer at a time, so keep a single connection to avoid SQLITE_BUSY.
	database.SetMaxOpenConns(1)
	if err := migrate(database); err != nil {
		return nil, err
	}
	return &Store{
		Books:     &SQLiteBookStore{database: database},
		Devices:   &SQLiteDeviceStore{database: database},
		Documents: &SQLiteDocumentStore{database: database},
		database:  database,
	}, nil
}

func (store *Store) Close() error {
	return store.database.Close()
}
