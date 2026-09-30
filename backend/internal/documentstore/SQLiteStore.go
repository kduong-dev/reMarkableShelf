package documentstore

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/kduong-dev/goutil/fatal"
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
		return err
	}
	defer tx.Rollback()

	for _, doc := range docs {
		_, err := tx.Exec(
			`INSERT INTO remarkable_documents (uuid, device_id, title, file_type, last_modified, linked_book_id, current_page, page_count, position_updated_at, book_title, book_author, parent_uuid, removed_at)
			 VALUES (?, ?, ?, ?, ?, NULL, ?, ?, ?, ?, ?, ?, NULL)
			 ON CONFLICT(uuid, device_id) DO UPDATE SET
			   parent_uuid = excluded.parent_uuid,
			   removed_at = NULL,
			   title = excluded.title,
			   file_type = excluded.file_type,
			   last_modified = excluded.last_modified,
			   current_page = excluded.current_page,
			   page_count = excluded.page_count,
			   position_updated_at = excluded.position_updated_at,
			   book_title = excluded.book_title,
			   book_author = excluded.book_author`,
			doc.UUID, deviceID, doc.Title, doc.FileType, doc.LastModified, doc.CurrentPage, doc.PageCount, doc.PositionUpdatedAt, doc.BookTitle, doc.BookAuthor, doc.ParentUUID,
		)
		if err != nil {
			return fmt.Errorf("upserting document %q: %w", doc.UUID, err)
		}
	}
	_, err = tx.Exec(
		`UPDATE remarkable_documents SET removed_at = ? WHERE device_id = ? AND removed_at IS NULL AND uuid NOT IN (SELECT value FROM json_each(?))`,
		time.Now().UTC(), deviceID, string(fatal.UnlessMarshal(uuidsOf(docs))),
	)
	if err != nil {
		return fmt.Errorf("marking removed documents: %w", err)
	}
	return tx.Commit()
}

func uuidsOf(docs []models.RemarkableDocument) []string {
	uuids := make([]string, 0, len(docs))
	for _, doc := range docs {
		uuids = append(uuids, doc.UUID)
	}
	return uuids
}

func (documentStore *SQLiteStore) ReplaceFolders(deviceID string, folders []models.RemarkableFolder) error {
	tx, err := documentStore.database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM remarkable_folders WHERE device_id = ?`, deviceID); err != nil {
		return fmt.Errorf("replacing folders: %w", err)
	}
	for _, folder := range folders {
		_, err := tx.Exec(`INSERT INTO remarkable_folders (uuid, device_id, title, parent_uuid) VALUES (?, ?, ?, ?)`,
			folder.UUID, deviceID, folder.Title, folder.ParentUUID)
		if err != nil {
			return fmt.Errorf("replacing folder %q: %w", folder.UUID, err)
		}
	}
	return tx.Commit()
}

func (documentStore *SQLiteStore) ListFolders(deviceID string) ([]models.RemarkableFolder, error) {
	rows, err := documentStore.database.Query(`SELECT uuid, title, parent_uuid FROM remarkable_folders WHERE device_id = ? ORDER BY title`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("listing folders: %w", err)
	}
	defer rows.Close()
	folders := []models.RemarkableFolder{}
	for rows.Next() {
		var folder models.RemarkableFolder
		if err := rows.Scan(&folder.UUID, &folder.Title, &folder.ParentUUID); err != nil {
			return nil, fmt.Errorf("listing folders: %w", err)
		}
		folders = append(folders, folder)
	}
	return folders, rows.Err()
}

func (documentStore *SQLiteStore) ListByDevice(deviceID string) ([]models.RemarkableDocument, error) {
	return documentStore.list(`device_id = ? AND removed_at IS NULL`, deviceID)
}

func (documentStore *SQLiteStore) ListByBook(bookID string) ([]models.RemarkableDocument, error) {
	return documentStore.list(`linked_book_id = ? AND removed_at IS NULL`, bookID)
}

func (documentStore *SQLiteStore) list(where, arg string) ([]models.RemarkableDocument, error) {
	rows, err := documentStore.database.Query(
		`SELECT uuid, device_id, title, file_type, last_modified, linked_book_id, current_page, page_count, position_updated_at, book_title, book_author, auto_link_dismissed, cover_image IS NOT NULL, cover_checked_at, parent_uuid, not_a_book
		 FROM remarkable_documents WHERE `+where+` ORDER BY last_modified DESC`,
		arg,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	docs := []models.RemarkableDocument{}
	for rows.Next() {
		var d models.RemarkableDocument
		if err := rows.Scan(&d.UUID, &d.DeviceID, &d.Title, &d.FileType, &d.LastModified, &d.LinkedBookID, &d.CurrentPage, &d.PageCount, &d.PositionUpdatedAt, &d.BookTitle, &d.BookAuthor, &d.AutoLinkDismissed, &d.HasCover, &d.CoverCheckedAt, &d.ParentUUID, &d.NotABook); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

func (documentStore *SQLiteStore) LinkToBook(deviceID, docUUID, bookID string) error {
	_, err := documentStore.database.Exec(
		`UPDATE remarkable_documents SET linked_book_id = ?, auto_link_dismissed = 0, not_a_book = 0 WHERE device_id = ? AND uuid = ?`,
		bookID, deviceID, docUUID,
	)
	return err
}

func (documentStore *SQLiteStore) SetNotABook(deviceID, docUUID string, notABook bool) error {
	result, err := documentStore.database.Exec(
		`UPDATE remarkable_documents SET not_a_book = ?, linked_book_id = CASE WHEN ? THEN NULL ELSE linked_book_id END
		 WHERE device_id = ? AND uuid = ? AND removed_at IS NULL`,
		notABook, notABook, deviceID, docUUID,
	)
	if err != nil {
		return fmt.Errorf("marking document: %w", err)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return fmt.Errorf("document %q on device %q: %w", docUUID, deviceID, ErrDocumentNotFound)
	}
	return nil
}

func (documentStore *SQLiteStore) UnlinkBook(bookID string) error {
	_, err := documentStore.database.Exec(
		`UPDATE remarkable_documents SET linked_book_id = NULL, auto_link_dismissed = 1 WHERE linked_book_id = ?`,
		bookID,
	)
	if err != nil {
		return fmt.Errorf("unlinking the book's documents: %w", err)
	}
	return nil
}

func (documentStore *SQLiteStore) Unlink(deviceID, docUUID string) error {
	result, err := documentStore.database.Exec(
		`UPDATE remarkable_documents SET linked_book_id = NULL, auto_link_dismissed = 1 WHERE device_id = ? AND uuid = ?`,
		deviceID, docUUID,
	)
	if err != nil {
		return fmt.Errorf("unlinking document: %w", err)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return fmt.Errorf("document %q on device %q: %w", docUUID, deviceID, ErrDocumentNotFound)
	}
	return nil
}

func (documentStore *SQLiteStore) SetCover(deviceID, docUUID string, image []byte, checkedAt time.Time) error {
	_, err := documentStore.database.Exec(
		`UPDATE remarkable_documents SET cover_image = COALESCE(?, cover_image), cover_checked_at = ? WHERE device_id = ? AND uuid = ?`,
		image, checkedAt, deviceID, docUUID,
	)
	return err
}

func (documentStore *SQLiteStore) GetCover(deviceID, docUUID string) ([]byte, error) {
	var image []byte
	err := documentStore.database.QueryRow(
		`SELECT cover_image FROM remarkable_documents WHERE device_id = ? AND uuid = ? AND cover_image IS NOT NULL`,
		deviceID, docUUID,
	).Scan(&image)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("cover of document %q on device %q: %w", docUUID, deviceID, ErrCoverNotFound)
	}
	return image, err
}
