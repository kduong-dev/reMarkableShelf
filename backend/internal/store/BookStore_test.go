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
		Convey("When changing only the page count of a bookmarked book", func() {
			_, err := opened.UpdateBook(book.ID, models.Book{CurrentPage: pages(300)})
			So(err, ShouldBeNil)
			updated, err := opened.UpdateBook(book.ID, models.Book{PageCount: pages(200)})
			Convey("Then the bookmark keeps its place, the same fraction through", func() {
				So(err, ShouldBeNil)
				So(*updated.CurrentPage, ShouldEqual, 110)
				So(*updated.PageCount, ShouldEqual, 200)
			})
		})
		Convey("When setting a page and a page count it's past the end of", func() {
			_, err := opened.UpdateBook(book.ID, models.Book{CurrentPage: pages(300), PageCount: pages(200)})
			Convey("Then it returns ErrCurrentPageOutOfRange", func() {
				So(merry.Is(err, store.ErrCurrentPageOutOfRange), ShouldBeTrue)
			})
		})
		Convey("When matching the book to an Open Library edition", func() {
			updated, err := opened.UpdateBook(book.ID, models.Book{Title: "Dune (Deluxe)", ISBN: "9780593099322", OpenLibraryID: "OL893415W"})
			Convey("Then its ISBN and Open Library ID are saved", func() {
				So(err, ShouldBeNil)
				stored, err := opened.GetBook(updated.ID)
				So(err, ShouldBeNil)
				So(stored.Title, ShouldEqual, "Dune (Deluxe)")
				So(stored.ISBN, ShouldEqual, "9780593099322")
				So(stored.OpenLibraryID, ShouldEqual, "OL893415W")
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
