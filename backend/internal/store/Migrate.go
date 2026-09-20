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
	google_books_id   TEXT NOT NULL DEFAULT '',
	created_at        DATETIME NOT NULL,
	updated_at        DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS devices (
	id              TEXT PRIMARY KEY,
	name            TEXT NOT NULL,
	host            TEXT NOT NULL,
	last_synced_at  DATETIME
);

CREATE TABLE IF NOT EXISTS remarkable_documents (
	uuid            TEXT NOT NULL,
	device_id       TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
	title           TEXT NOT NULL,
	file_type       TEXT NOT NULL,
	last_modified   DATETIME,
	linked_book_id  TEXT REFERENCES books(id) ON DELETE SET NULL,
	PRIMARY KEY (uuid, device_id)
);

CREATE INDEX IF NOT EXISTS idx_remarkable_documents_device ON remarkable_documents(device_id);
`

func (s *Store) migrate() error {
	_, err := s.DB.Exec(schema)
	if err != nil {
		return merry.Wrap(err).WithUserMessage("running database migrations")
	}
	return nil
}
