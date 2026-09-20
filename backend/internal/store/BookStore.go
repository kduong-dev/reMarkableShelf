package store

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/ansel1/merry"
	"github.com/google/uuid"
	"github.com/kqvd/reMarkableShelf/backend/internal/models"
)

func (s *Store) ListBooks() ([]models.Book, error) {
	rows, err := s.DB.Query(`SELECT id, title, author, isbn, cover_url, status, rating, source, google_books_id, created_at, updated_at FROM books ORDER BY updated_at DESC`)
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
	row := s.DB.QueryRow(`SELECT id, title, author, isbn, cover_url, status, rating, source, google_books_id, created_at, updated_at FROM books WHERE id = ?`, id)
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
	now := time.Now().UTC()
	book.CreatedAt, book.UpdatedAt = now, now

	_, err := s.DB.Exec(
		`INSERT INTO books (id, title, author, isbn, cover_url, status, rating, source, google_books_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		book.ID, book.Title, book.Author, book.ISBN, book.CoverURL, book.Status, book.Rating, book.Source, book.GoogleBooksID, book.CreatedAt, book.UpdatedAt,
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
	existing.UpdatedAt = time.Now().UTC()

	_, err = s.DB.Exec(
		`UPDATE books SET title = ?, author = ?, isbn = ?, cover_url = ?, status = ?, rating = ?, updated_at = ? WHERE id = ?`,
		existing.Title, existing.Author, existing.ISBN, existing.CoverURL, existing.Status, existing.Rating, existing.UpdatedAt, existing.ID,
	)
	if err != nil {
		return models.Book{}, merry.Wrap(err).WithUserMessage("updating book")
	}
	return existing, nil
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

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBook(row rowScanner) (models.Book, error) {
	var b models.Book
	err := row.Scan(&b.ID, &b.Title, &b.Author, &b.ISBN, &b.CoverURL, &b.Status, &b.Rating, &b.Source, &b.GoogleBooksID, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Book{}, err
		}
		return models.Book{}, merry.Wrap(err)
	}
	return b, nil
}
