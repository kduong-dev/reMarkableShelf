package database

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

// Open opens the sqlite database at path and migrates it to the current
// schema.
func Open(path string) (*sql.DB, error) {
	// SQLite ignores the schema's REFERENCES clauses, including ON DELETE
	// CASCADE and SET NULL, unless foreign keys are switched on per
	// connection, which the DSN does for every connection the pool opens.
	database, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	if err := database.Ping(); err != nil {
		return nil, err
	}
	// xochitl syncs can upsert many rows at once; sqlite only allows one
	// writer at a time, so keep a single connection to avoid SQLITE_BUSY.
	database.SetMaxOpenConns(1)
	if err := migrate(database); err != nil {
		_ = database.Close()
		return nil, err
	}
	return database, nil
}
