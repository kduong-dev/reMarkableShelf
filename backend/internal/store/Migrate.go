package store

import "github.com/ansel1/merry"

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
	paired_at       DATETIME
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
	PRIMARY KEY (uuid, device_id)
);

CREATE INDEX IF NOT EXISTS idx_remarkable_documents_device ON remarkable_documents(device_id);
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
}

func (s *Store) migrate() error {
	if err := s.renameGoogleBooksIDColumn(); err != nil {
		return err
	}
	_, err := s.DB.Exec(schema)
	if err != nil {
		return merry.Wrap(err).WithUserMessage("running database migrations")
	}
	for _, column := range addedColumns {
		if err := s.addColumnIfMissing(column.table, column.name, column.definition); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) columnExists(table, column string) (bool, error) {
	var count int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&count)
	if err != nil {
		return false, merry.Wrap(err).WithUserMessagef("inspecting %s table", table)
	}
	return count > 0, nil
}

// addColumnIfMissing brings tables created by an older schema up to date,
// since CREATE TABLE IF NOT EXISTS leaves existing tables untouched.
func (s *Store) addColumnIfMissing(table, column, definition string) error {
	exists, err := s.columnExists(table, column)
	if err != nil || exists {
		return err
	}
	_, err = s.DB.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + definition)
	if err != nil {
		return merry.Wrap(err).WithUserMessagef("adding %s.%s column", table, column)
	}
	return nil
}

// renameGoogleBooksIDColumn upgrades databases created before search moved
// from Google Books to Open Library. The stored Google volume IDs mean nothing
// to Open Library, so they are cleared rather than carried over.
func (s *Store) renameGoogleBooksIDColumn() error {
	exists, err := s.columnExists("books", "google_books_id")
	if err != nil || !exists {
		return err
	}
	_, err = s.DB.Exec(`ALTER TABLE books RENAME COLUMN google_books_id TO open_library_id; UPDATE books SET open_library_id = ''`)
	if err != nil {
		return merry.Wrap(err).WithUserMessage("renaming google_books_id column")
	}
	return nil
}
