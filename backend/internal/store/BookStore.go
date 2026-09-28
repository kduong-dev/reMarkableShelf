package store

import (
	"time"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

// BookStore holds the books on the user's shelf. Each book also carries the
// cover and page count of any tablet document linked to it.
type BookStore interface {
	List() ([]models.Book, error)
	Get(id string) (models.Book, error)
	// Create inserts a new book, generating an id/timestamps if unset.
	Create(book models.Book) (models.Book, error)
	// Update replaces the mutable fields of an existing book (status, rating, etc).
	Update(id string, patch models.Book) (models.Book, error)
	// SetTabletProgress moves a book's bookmark to a position synced from a
	// tablet, stamping it with the tablet's time rather than now so that a
	// later change in the app still counts as newer.
	SetTabletProgress(input SetTabletProgressInput) (models.Book, error)
	Delete(id string) error
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
