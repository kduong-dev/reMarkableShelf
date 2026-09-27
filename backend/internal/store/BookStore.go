package store

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/ansel1/merry"
	"github.com/google/uuid"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

// selectBooks reads every book column plus the cover path of one linked
// tablet document with a synced cover, which the API serves at
// /api/devices/{device}/documents/{uuid}/cover.
const selectBooks = `SELECT id, title, author, isbn, cover_url, status, rating, source, open_library_id, page_count, current_page, progress_updated_at, progress_source, created_at, updated_at,
	(SELECT '/api/devices/' || device_id || '/documents/' || uuid || '/cover' FROM remarkable_documents
	 WHERE linked_book_id = books.id AND cover_image IS NOT NULL LIMIT 1)
	FROM books`

func (s *Store) ListBooks() ([]models.Book, error) {
	rows, err := s.DB.Query(selectBooks + ` ORDER BY updated_at DESC`)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	defer rows.Close()

	books := []models.Book{}
	for rows.Next() {
		book, err := scanBook(rows)
		if err != nil {
			return nil, err
		}
		books = append(books, book)
	}
	return books, merry.Wrap(rows.Err())
}

func (s *Store) GetBook(id string) (models.Book, error) {
	row := s.DB.QueryRow(selectBooks+` WHERE id = ?`, id)
	book, err := scanBook(row)
	if err == sql.ErrNoRows {
		return models.Book{}, merry.New("book not found").WithHTTPCode(http.StatusNotFound).WithUserMessagef("no book with id %q", id)
	}
	return book, err
}

// CreateBook inserts a new book, generating an id/timestamps if unset.
func (s *Store) CreateBook(book models.Book) (models.Book, error) {
	if book.ID == "" {
		book.ID = uuid.NewString()
	}
	if book.Status == "" {
		book.Status = models.StatusWantToRead
	}
	if book.Source == "" {
		book.Source = models.SourceManual
	}
	if err := validateProgress(book); err != nil {
		return models.Book{}, err
	}
	now := time.Now().UTC()
	book.CreatedAt, book.UpdatedAt = now, now
	book.ProgressUpdatedAt, book.ProgressSource = nil, ""
	if book.CurrentPage != nil {
		book.ProgressUpdatedAt, book.ProgressSource = &now, models.ProgressSourceApp
	}

	_, err := s.DB.Exec(
		`INSERT INTO books (id, title, author, isbn, cover_url, status, rating, source, open_library_id, page_count, current_page, progress_updated_at, progress_source, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		book.ID, book.Title, book.Author, book.ISBN, book.CoverURL, book.Status, book.Rating, book.Source, book.OpenLibraryID, book.PageCount, book.CurrentPage, book.ProgressUpdatedAt, book.ProgressSource, book.CreatedAt, book.UpdatedAt,
	)
	if err != nil {
		return models.Book{}, merry.Wrap(err).WithUserMessage("creating book")
	}
	return book, nil
}

// UpdateBook replaces the mutable fields of an existing book (status, rating, etc).
func (s *Store) UpdateBook(id string, patch models.Book) (models.Book, error) {
	existing, err := s.GetBook(id)
	if err != nil {
		return models.Book{}, err
	}
	if patch.Title != "" {
		existing.Title = patch.Title
	}
	if patch.Author != "" {
		existing.Author = patch.Author
	}
	if patch.Status != "" {
		existing.Status = patch.Status
	}
	if patch.Rating != nil {
		existing.Rating = patch.Rating
	}
	if patch.CoverURL != "" {
		existing.CoverURL = patch.CoverURL
	}
	if patch.ISBN != "" {
		existing.ISBN = patch.ISBN
	}
	if patch.OpenLibraryID != "" {
		existing.OpenLibraryID = patch.OpenLibraryID
	}
	if patch.PageCount != nil {
		// A new page count on its own, e.g. from matching the book to an
		// edition, keeps the bookmark the same fraction of the way through.
		if patch.CurrentPage == nil && existing.CurrentPage != nil && existing.PageCount != nil {
			scaled := models.ScalePage(*existing.CurrentPage, *existing.PageCount, *patch.PageCount)
			existing.CurrentPage = &scaled
		}
		existing.PageCount = patch.PageCount
	}
	existing.UpdatedAt = time.Now().UTC()
	if patch.CurrentPage != nil {
		existing.CurrentPage = patch.CurrentPage
		existing.ProgressUpdatedAt, existing.ProgressSource = &existing.UpdatedAt, models.ProgressSourceApp
	}
	if err := validateProgress(existing); err != nil {
		return models.Book{}, err
	}

	_, err = s.DB.Exec(
		`UPDATE books SET title = ?, author = ?, isbn = ?, cover_url = ?, open_library_id = ?, status = ?, rating = ?, page_count = ?, current_page = ?, progress_updated_at = ?, progress_source = ?, updated_at = ? WHERE id = ?`,
		existing.Title, existing.Author, existing.ISBN, existing.CoverURL, existing.OpenLibraryID, existing.Status, existing.Rating, existing.PageCount, existing.CurrentPage, existing.ProgressUpdatedAt, existing.ProgressSource, existing.UpdatedAt, existing.ID,
	)
	if err != nil {
		return models.Book{}, merry.Wrap(err).WithUserMessage("updating book")
	}
	return existing, nil
}

// SetTabletProgressInput is a reading position taken from a tablet, already
// mapped onto the book's pages. UpdatedAt is when the tablet recorded it.
type SetTabletProgressInput struct {
	BookID      string
	CurrentPage int
	PageCount   int
	Status      models.BookStatus
	UpdatedAt   time.Time
}

// SetTabletProgress moves a book's bookmark to a position synced from a
// tablet, stamping it with the tablet's time rather than now so that a later
// change in the app still counts as newer.
func (s *Store) SetTabletProgress(input SetTabletProgressInput) (models.Book, error) {
	book, err := s.GetBook(input.BookID)
	if err != nil {
		return models.Book{}, err
	}
	book.CurrentPage, book.PageCount, book.Status = &input.CurrentPage, &input.PageCount, input.Status
	book.ProgressUpdatedAt, book.ProgressSource = &input.UpdatedAt, models.ProgressSourceRemarkable
	book.UpdatedAt = time.Now().UTC()
	if err := validateProgress(book); err != nil {
		return models.Book{}, err
	}
	_, err = s.DB.Exec(
		`UPDATE books SET status = ?, page_count = ?, current_page = ?, progress_updated_at = ?, progress_source = ?, updated_at = ? WHERE id = ?`,
		book.Status, book.PageCount, book.CurrentPage, book.ProgressUpdatedAt, book.ProgressSource, book.UpdatedAt, book.ID,
	)
	if err != nil {
		return models.Book{}, merry.Wrap(err).WithUserMessage("updating book progress")
	}
	return book, nil
}

func (s *Store) DeleteBook(id string) error {
	res, err := s.DB.Exec(`DELETE FROM books WHERE id = ?`, id)
	if err != nil {
		return merry.Wrap(err).WithUserMessage("deleting book")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return merry.New("book not found").WithHTTPCode(http.StatusNotFound).WithUserMessagef("no book with id %q", id)
	}
	return nil
}

// validateProgress rejects a bookmark that can't exist: negative numbers, or
// a current page past the end of a book whose length is known.
func validateProgress(book models.Book) error {
	if book.PageCount != nil && *book.PageCount < 1 {
		return merry.Here(ErrInvalidPageCount)
	}
	if book.CurrentPage != nil && *book.CurrentPage < 0 {
		return merry.Here(ErrCurrentPageOutOfRange)
	}
	if book.PageCount != nil && book.CurrentPage != nil && *book.CurrentPage > *book.PageCount {
		return merry.Here(ErrCurrentPageOutOfRange).WithUserMessagef("page %d is past the end of a %d-page book", *book.CurrentPage, *book.PageCount)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBook(row rowScanner) (models.Book, error) {
	var b models.Book
	var tabletCoverURL sql.NullString
	err := row.Scan(&b.ID, &b.Title, &b.Author, &b.ISBN, &b.CoverURL, &b.Status, &b.Rating, &b.Source, &b.OpenLibraryID, &b.PageCount, &b.CurrentPage, &b.ProgressUpdatedAt, &b.ProgressSource, &b.CreatedAt, &b.UpdatedAt, &tabletCoverURL)
	b.TabletCoverURL = tabletCoverURL.String
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Book{}, err
		}
		return models.Book{}, merry.Wrap(err)
	}
	return b, nil
}
