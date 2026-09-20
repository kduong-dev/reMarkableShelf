package store

import (
	"database/sql"

	"github.com/ansel1/merry"
	_ "modernc.org/sqlite"
)

// Store wraps the sqlite connection shared by all the *Store method files
// in this package (BookStore.go, DeviceStore.go, DocumentStore.go).
type Store struct {
	DB *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	if err := db.Ping(); err != nil {
		return nil, merry.Wrap(err)
	}
	// xochitl syncs can upsert many rows at once; sqlite only allows one
	// writer at a time, so keep a single connection to avoid SQLITE_BUSY.
	db.SetMaxOpenConns(1)
	s := &Store{DB: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.DB.Close()
}
