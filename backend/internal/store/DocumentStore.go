package store

import (
	"net/http"

	"github.com/ansel1/merry"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

// UpsertDocuments replaces the known document set for a device with the
// freshly-synced list, preserving any existing linked_book_id.
func (s *Store) UpsertDocuments(deviceID string, docs []models.RemarkableDocument) error {
	tx, err := s.DB.Begin()
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

func (s *Store) ListDocumentsByDevice(deviceID string) ([]models.RemarkableDocument, error) {
	rows, err := s.DB.Query(
		`SELECT uuid, device_id, title, file_type, last_modified, linked_book_id, current_page, page_count, position_updated_at, book_title, book_author, auto_link_dismissed
		 FROM remarkable_documents WHERE device_id = ? ORDER BY last_modified DESC`,
		deviceID,
	)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	defer rows.Close()

	docs := []models.RemarkableDocument{}
	for rows.Next() {
		var d models.RemarkableDocument
		if err := rows.Scan(&d.UUID, &d.DeviceID, &d.Title, &d.FileType, &d.LastModified, &d.LinkedBookID, &d.CurrentPage, &d.PageCount, &d.PositionUpdatedAt, &d.BookTitle, &d.BookAuthor, &d.AutoLinkDismissed); err != nil {
			return nil, merry.Wrap(err)
		}
		docs = append(docs, d)
	}
	return docs, merry.Wrap(rows.Err())
}

func (s *Store) LinkDocumentToBook(deviceID, docUUID, bookID string) error {
	_, err := s.DB.Exec(
		`UPDATE remarkable_documents SET linked_book_id = ?, auto_link_dismissed = 0 WHERE device_id = ? AND uuid = ?`,
		bookID, deviceID, docUUID,
	)
	return merry.Wrap(err)
}

// UnlinkDocument detaches a document from its book and stops sync linking
// it again by title.
func (s *Store) UnlinkDocument(deviceID, docUUID string) error {
	result, err := s.DB.Exec(
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
