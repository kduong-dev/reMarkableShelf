package documentstore

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/ansel1/merry"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

type SQLiteStore struct {
	database *sql.DB
}

func NewSQLiteStore(database *sql.DB) *SQLiteStore {
	return &SQLiteStore{database: database}
}

func (documentStore *SQLiteStore) Upsert(deviceID string, docs []models.RemarkableDocument) error {
	tx, err := documentStore.database.Begin()
	if err != nil {
		return merry.Wrap(err)
	}
	defer tx.Rollback()

	for _, doc := range docs {
		_, err := tx.Exec(
			`INSERT INTO remarkable_documents (uuid, device_id, title, file_type, last_modified, linked_book_id, current_page, page_count, position_updated_at, book_title, book_author)
			 VALUES (?, ?, ?, ?, ?, NULL, ?, ?, ?, ?, ?)
			 ON CONFLICT(uuid, device_id) DO UPDATE SET
			   title = excluded.title,
			   file_type = excluded.file_type,
			   last_modified = excluded.last_modified,
			   current_page = excluded.current_page,
			   page_count = excluded.page_count,
			   position_updated_at = excluded.position_updated_at,
			   book_title = excluded.book_title,
			   book_author = excluded.book_author`,
			doc.UUID, deviceID, doc.Title, doc.FileType, doc.LastModified, doc.CurrentPage, doc.PageCount, doc.PositionUpdatedAt, doc.BookTitle, doc.BookAuthor,
		)
		if err != nil {
			return merry.Wrap(err).WithUserMessagef("upserting document %q", doc.UUID)
		}
	}
	return merry.Wrap(tx.Commit())
}

func (documentStore *SQLiteStore) ListByDevice(deviceID string) ([]models.RemarkableDocument, error) {
	return documentStore.list(`device_id = ?`, deviceID)
}

func (documentStore *SQLiteStore) ListByBook(bookID string) ([]models.RemarkableDocument, error) {
	return documentStore.list(`linked_book_id = ?`, bookID)
}

func (documentStore *SQLiteStore) list(where, arg string) ([]models.RemarkableDocument, error) {
	rows, err := documentStore.database.Query(
		`SELECT uuid, device_id, title, file_type, last_modified, linked_book_id, current_page, page_count, position_updated_at, book_title, book_author, auto_link_dismissed, cover_image IS NOT NULL, cover_checked_at
		 FROM remarkable_documents WHERE `+where+` ORDER BY last_modified DESC`,
		arg,
	)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	defer rows.Close()

	docs := []models.RemarkableDocument{}
	for rows.Next() {
		var d models.RemarkableDocument
		if err := rows.Scan(&d.UUID, &d.DeviceID, &d.Title, &d.FileType, &d.LastModified, &d.LinkedBookID, &d.CurrentPage, &d.PageCount, &d.PositionUpdatedAt, &d.BookTitle, &d.BookAuthor, &d.AutoLinkDismissed, &d.HasCover, &d.CoverCheckedAt); err != nil {
			return nil, merry.Wrap(err)
		}
		docs = append(docs, d)
	}
	return docs, merry.Wrap(rows.Err())
}

func (documentStore *SQLiteStore) LinkToBook(deviceID, docUUID, bookID string) error {
	_, err := documentStore.database.Exec(
		`UPDATE remarkable_documents SET linked_book_id = ?, auto_link_dismissed = 0 WHERE device_id = ? AND uuid = ?`,
		bookID, deviceID, docUUID,
	)
	return merry.Wrap(err)
}

func (documentStore *SQLiteStore) Unlink(deviceID, docUUID string) error {
	result, err := documentStore.database.Exec(
		`UPDATE remarkable_documents SET linked_book_id = NULL, auto_link_dismissed = 1 WHERE device_id = ? AND uuid = ?`,
		deviceID, docUUID,
	)
	if err != nil {
		return merry.Wrap(err).WithUserMessage("unlinking document")
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return merry.New("document not found").WithHTTPCode(http.StatusNotFound).WithUserMessagef("no document %q on that device", docUUID)
	}
	return nil
}

func (documentStore *SQLiteStore) SetCover(deviceID, docUUID string, image []byte, checkedAt time.Time) error {
	_, err := documentStore.database.Exec(
		`UPDATE remarkable_documents SET cover_image = COALESCE(?, cover_image), cover_checked_at = ? WHERE device_id = ? AND uuid = ?`,
		image, checkedAt, deviceID, docUUID,
	)
	return merry.Wrap(err)
}

func (documentStore *SQLiteStore) GetCover(deviceID, docUUID string) ([]byte, error) {
	var image []byte
	err := documentStore.database.QueryRow(
		`SELECT cover_image FROM remarkable_documents WHERE device_id = ? AND uuid = ? AND cover_image IS NOT NULL`,
		deviceID, docUUID,
	).Scan(&image)
	if err == sql.ErrNoRows {
		return nil, merry.New("cover not found").WithHTTPCode(http.StatusNotFound).WithUserMessage("that document has no synced cover")
	}
	return image, merry.Wrap(err)
}
