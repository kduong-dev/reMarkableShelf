package store_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	_ "modernc.org/sqlite"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/store"
)

func TestMigrate(t *testing.T) {
	Convey("Given a database created while search used Google Books", t, func() {
		path := filepath.Join(t.TempDir(), "legacy.db")
		legacy, err := sql.Open("sqlite", path)
		So(err, ShouldBeNil)
		_, err = legacy.Exec(`CREATE TABLE books (
			id TEXT PRIMARY KEY, title TEXT NOT NULL, author TEXT NOT NULL DEFAULT '',
			isbn TEXT NOT NULL DEFAULT '', cover_url TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'want_to_read', rating INTEGER,
			source TEXT NOT NULL DEFAULT 'manual', google_books_id TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL);
			INSERT INTO books (id, title, google_books_id, created_at, updated_at)
			VALUES ('book-1', 'Dune', 'abc123', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
		So(err, ShouldBeNil)
		So(legacy.Close(), ShouldBeNil)
		Convey("When the store opens it", func() {
			opened, err := store.Open(path)
			So(err, ShouldBeNil)
			Reset(func() { _ = opened.Close() })
			Convey("Then the page columns are added", func() {
				pageCount := 544
				_, err := opened.UpdateBook("book-1", models.Book{PageCount: &pageCount})
				So(err, ShouldBeNil)
			})
			Convey("Then the book is kept with its Google Books ID cleared", func() {
				book, err := opened.GetBook("book-1")
				So(err, ShouldBeNil)
				So(book.Title, ShouldEqual, "Dune")
				So(book.OpenLibraryID, ShouldBeEmpty)
			})
			Convey("Then opening it again is a no-op", func() {
				So(opened.Close(), ShouldBeNil)
				reopened, err := store.Open(path)
				So(err, ShouldBeNil)
				So(reopened.Close(), ShouldBeNil)
			})
		})
	})
}
