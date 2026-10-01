package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

const schema = `
CREATE TABLE IF NOT EXISTS books (
	id                TEXT PRIMARY KEY,
	title             TEXT NOT NULL,
	author            TEXT NOT NULL DEFAULT '',
	isbn              TEXT NOT NULL DEFAULT '',
	cover_url         TEXT NOT NULL DEFAULT '',
	status            TEXT NOT NULL DEFAULT 'want_to_read',
	rating            INTEGER,
	source            TEXT NOT NULL DEFAULT 'manual',
	open_library_id   TEXT NOT NULL DEFAULT '',
	page_count        INTEGER,
	current_page      INTEGER,
	progress_updated_at DATETIME,
	progress_source   TEXT NOT NULL DEFAULT '',
	created_at        DATETIME NOT NULL,
	updated_at        DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS devices (
	id              TEXT PRIMARY KEY,
	name            TEXT NOT NULL,
	host            TEXT NOT NULL,
	last_synced_at  DATETIME,
	paired_at       DATETIME,
	host_key        TEXT NOT NULL DEFAULT '',
	identity_changed INTEGER NOT NULL DEFAULT 0,
	downloads_folder_uuid TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS remarkable_documents (
	uuid            TEXT NOT NULL,
	device_id       TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
	title           TEXT NOT NULL,
	file_type       TEXT NOT NULL,
	last_modified   DATETIME,
	linked_book_id  TEXT REFERENCES books(id) ON DELETE SET NULL,
	current_page    INTEGER,
	page_count      INTEGER,
	position_updated_at DATETIME,
	book_title      TEXT NOT NULL DEFAULT '',
	book_author     TEXT NOT NULL DEFAULT '',
	auto_link_dismissed INTEGER NOT NULL DEFAULT 0,
	cover_image     BLOB,
	cover_checked_at DATETIME,
	parent_uuid     TEXT NOT NULL DEFAULT '',
	removed_at      DATETIME,
	not_a_book      INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (uuid, device_id)
);

CREATE TABLE IF NOT EXISTS remarkable_folders (
	uuid            TEXT NOT NULL,
	device_id       TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
	title           TEXT NOT NULL,
	parent_uuid     TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (uuid, device_id)
);

CREATE INDEX IF NOT EXISTS idx_remarkable_documents_device ON remarkable_documents(device_id);

CREATE TABLE IF NOT EXISTS ebook_sources (
	id              TEXT PRIMARY KEY,
	kind            TEXT NOT NULL,
	name            TEXT NOT NULL,
	url             TEXT NOT NULL DEFAULT '',
	search_url      TEXT NOT NULL DEFAULT '',
	priority        INTEGER NOT NULL,
	enabled         INTEGER NOT NULL DEFAULT 1,
	created_at      DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS book_files (
	book_id         TEXT PRIMARY KEY REFERENCES books(id) ON DELETE CASCADE,
	storage_file_id TEXT NOT NULL,
	format          TEXT NOT NULL,
	size            INTEGER NOT NULL,
	source          TEXT NOT NULL,
	source_name     TEXT NOT NULL DEFAULT '',
	saved_at        DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS book_file_deliveries (
	book_id         TEXT NOT NULL REFERENCES book_files(book_id) ON DELETE CASCADE,
	device_id       TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
	document_uuid   TEXT NOT NULL,
	copied_at       DATETIME NOT NULL,
	loaded_at       DATETIME,
	PRIMARY KEY (book_id, device_id)
);
`

// addedColumns are the columns added to the schema after its tables were
// first created, in the order they were added.
var addedColumns = []struct {
	table      string
	name       string
	definition string
}{
	{"books", "page_count", "INTEGER"},
	{"books", "current_page", "INTEGER"},
	{"books", "progress_updated_at", "DATETIME"},
	{"books", "progress_source", "TEXT NOT NULL DEFAULT ''"},
	{"remarkable_documents", "current_page", "INTEGER"},
	{"remarkable_documents", "page_count", "INTEGER"},
	{"remarkable_documents", "position_updated_at", "DATETIME"},
	{"devices", "paired_at", "DATETIME"},
	{"remarkable_documents", "book_title", "TEXT NOT NULL DEFAULT ''"},
	{"remarkable_documents", "book_author", "TEXT NOT NULL DEFAULT ''"},
	{"remarkable_documents", "auto_link_dismissed", "INTEGER NOT NULL DEFAULT 0"},
	{"remarkable_documents", "cover_image", "BLOB"},
	{"remarkable_documents", "cover_checked_at", "DATETIME"},
	{"devices", "host_key", "TEXT NOT NULL DEFAULT ''"},
	{"devices", "identity_changed", "INTEGER NOT NULL DEFAULT 0"},
	{"remarkable_documents", "parent_uuid", "TEXT NOT NULL DEFAULT ''"},
	{"remarkable_documents", "removed_at", "DATETIME"},
	{"devices", "downloads_folder_uuid", "TEXT NOT NULL DEFAULT ''"},
	{"remarkable_documents", "not_a_book", "INTEGER NOT NULL DEFAULT 0"},
	{"book_files", "source_name", "TEXT NOT NULL DEFAULT ''"},
}

func migrate(database *sql.DB) error {
	if err := renameGoogleBooksIDColumn(database); err != nil {
		return err
	}
	_, err := database.Exec(schema)
	if err != nil {
		return fmt.Errorf("running database migrations: %w", err)
	}
	for _, column := range addedColumns {
		if err := addColumnIfMissing(database, column.table, column.name, column.definition); err != nil {
			return err
		}
	}
	if err := seedEbookSources(database); err != nil {
		return err
	}
	return removeOrphans(database)
}

// seedEbookSources adds the built-in Internet Archive source, first in
// line, to a library that doesn't have it yet.
func seedEbookSources(database *sql.DB) error {
	_, err := database.Exec(
		`INSERT OR IGNORE INTO ebook_sources (id, kind, name, priority, enabled, created_at) VALUES (?, ?, ?, 0, 1, ?)`,
		models.BuiltInEbookSourceID, models.EbookSourceKindInternetArchive, "Internet Archive", time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("adding the built-in book source: %w", err)
	}
	return nil
}

// removeOrphans repairs rows left behind while foreign keys weren't
// enforced: documents linked to deleted books, and documents of deleted
// devices.
func removeOrphans(database *sql.DB) error {
	_, err := database.Exec(`
		UPDATE remarkable_documents SET linked_book_id = NULL
		 WHERE linked_book_id IS NOT NULL AND linked_book_id NOT IN (SELECT id FROM books);
		DELETE FROM remarkable_documents WHERE device_id NOT IN (SELECT id FROM devices);`)
	if err != nil {
		return fmt.Errorf("removing orphaned documents: %w", err)
	}
	return nil
}

func columnExists(database *sql.DB, table, column string) (bool, error) {
	var count int
	err := database.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("inspecting %s table: %w", table, err)
	}
	return count > 0, nil
}

// addColumnIfMissing brings tables created by an older schema up to date,
// since CREATE TABLE IF NOT EXISTS leaves existing tables untouched.
func addColumnIfMissing(database *sql.DB, table, column, definition string) error {
	exists, err := columnExists(database, table, column)
	if err != nil || exists {
		return err
	}
	_, err = database.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + definition)
	if err != nil {
		return fmt.Errorf("adding %s.%s column: %w", table, column, err)
	}
	return nil
}

// renameGoogleBooksIDColumn upgrades databases created before search moved
// from Google Books to Open Library. The stored Google volume IDs mean nothing
// to Open Library, so they are cleared rather than carried over.
func renameGoogleBooksIDColumn(database *sql.DB) error {
	exists, err := columnExists(database, "books", "google_books_id")
	if err != nil || !exists {
		return err
	}
	_, err = database.Exec(`ALTER TABLE books RENAME COLUMN google_books_id TO open_library_id; UPDATE books SET open_library_id = ''`)
	if err != nil {
		return fmt.Errorf("renaming google_books_id column: %w", err)
	}
	return nil
}
