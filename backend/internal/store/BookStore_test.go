package store_test

import (
	"path/filepath"
	"testing"

	"github.com/ansel1/merry"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/store"
)

func pages(count int) *int {
	return &count
}

func TestBookProgress(t *testing.T) {
	Convey("Given a store holding a 544-page book", t, func() {
		opened, err := store.Open(filepath.Join(t.TempDir(), "books.db"))
		So(err, ShouldBeNil)
		Reset(func() { _ = opened.Close() })
		book, err := opened.CreateBook(models.Book{Title: "Dune", PageCount: pages(544)})
		So(err, ShouldBeNil)
		Convey("When bookmarking a page within the book", func() {
			updated, err := opened.UpdateBook(book.ID, models.Book{CurrentPage: pages(120)})
			Convey("Then the bookmark is saved and the page count kept", func() {
				So(err, ShouldBeNil)
				So(*updated.CurrentPage, ShouldEqual, 120)
				stored, err := opened.GetBook(book.ID)
				So(err, ShouldBeNil)
				So(*stored.CurrentPage, ShouldEqual, 120)
				So(*stored.PageCount, ShouldEqual, 544)
			})
		})
		Convey("When bookmarking a page past the end", func() {
			_, err := opened.UpdateBook(book.ID, models.Book{CurrentPage: pages(545)})
			Convey("Then it returns ErrCurrentPageOutOfRange", func() {
				So(merry.Is(err, store.ErrCurrentPageOutOfRange), ShouldBeTrue)
			})
		})
		Convey("When bookmarking a negative page", func() {
			_, err := opened.UpdateBook(book.ID, models.Book{CurrentPage: pages(-1)})
			Convey("Then it returns ErrCurrentPageOutOfRange", func() {
				So(merry.Is(err, store.ErrCurrentPageOutOfRange), ShouldBeTrue)
			})
		})
		Convey("When shrinking the page count below the bookmark", func() {
			_, err := opened.UpdateBook(book.ID, models.Book{CurrentPage: pages(300)})
			So(err, ShouldBeNil)
			_, err = opened.UpdateBook(book.ID, models.Book{PageCount: pages(200)})
			Convey("Then it returns ErrCurrentPageOutOfRange", func() {
				So(merry.Is(err, store.ErrCurrentPageOutOfRange), ShouldBeTrue)
			})
		})
		Convey("When setting a page count of zero", func() {
			_, err := opened.UpdateBook(book.ID, models.Book{PageCount: pages(0)})
			Convey("Then it returns ErrInvalidPageCount", func() {
				So(merry.Is(err, store.ErrInvalidPageCount), ShouldBeTrue)
			})
		})
	})
	Convey("Given a store holding a book with an unknown page count", t, func() {
		opened, err := store.Open(filepath.Join(t.TempDir(), "books.db"))
		So(err, ShouldBeNil)
		Reset(func() { _ = opened.Close() })
		book, err := opened.CreateBook(models.Book{Title: "Meeting notes"})
		So(err, ShouldBeNil)
		Convey("When bookmarking any page", func() {
			updated, err := opened.UpdateBook(book.ID, models.Book{CurrentPage: pages(80)})
			Convey("Then the bookmark is saved without a page count", func() {
				So(err, ShouldBeNil)
				So(*updated.CurrentPage, ShouldEqual, 80)
				So(updated.PageCount, ShouldBeNil)
			})
		})
	})
}
