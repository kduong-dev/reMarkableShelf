package devicesync_test

import (
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicesync"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/store"
)

// fakeTablet serves whatever documents the test has put on it.
type fakeTablet struct {
	documents []models.RemarkableDocument
}

func (tablet *fakeTablet) ListDocuments(host string) ([]models.RemarkableDocument, error) {
	return tablet.documents, nil
}

func pages(count int) *int {
	return &count
}

// openedDocument is a PDF titled title, left on page currentPage of
// pageCount at openedAt.
func openedDocument(title string, currentPage, pageCount int, openedAt time.Time) models.RemarkableDocument {
	return models.RemarkableDocument{
		UUID:              "document-1",
		Title:             title,
		FileType:          models.FileTypePDF,
		LastModified:      openedAt,
		CurrentPage:       pages(currentPage),
		PageCount:         pages(pageCount),
		PositionUpdatedAt: &openedAt,
	}
}

func TestSyncDevice(t *testing.T) {
	Convey("Given a registered tablet and a library", t, func() {
		opened, err := store.Open(filepath.Join(t.TempDir(), "books.db"))
		So(err, ShouldBeNil)
		Reset(func() { _ = opened.Close() })
		device, err := opened.CreateDevice(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		tablet := &fakeTablet{}
		syncer := devicesync.NewSyncer(opened, tablet)
		sync := func() models.Book {
			_, err := syncer.SyncDevice(device.ID)
			So(err, ShouldBeNil)
			books, err := opened.ListBooks()
			So(err, ShouldBeNil)
			return books[0]
		}
		// An hour ago, at the millisecond precision the tablet records.
		tabletTime := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)

		Convey("When a 544-page book's PDF is on page 42 of 171 on the tablet", func() {
			_, err := opened.CreateBook(models.Book{Title: "Dune", PageCount: pages(544)})
			So(err, ShouldBeNil)
			tablet.documents = []models.RemarkableDocument{openedDocument("Dune", 42, 171, tabletTime)}
			book := sync()
			Convey("Then the document is linked by title and the bookmark moves the same fraction through the book", func() {
				So(*book.CurrentPage, ShouldEqual, 134)
				So(*book.PageCount, ShouldEqual, 544)
				So(book.Status, ShouldEqual, models.StatusReading)
				So(book.ProgressSource, ShouldEqual, models.ProgressSourceRemarkable)
				So(*book.ProgressUpdatedAt, ShouldEqual, tabletTime)
			})
			Convey("Then the tablet's own position is kept on the document", func() {
				documents, err := opened.ListDocumentsByDevice(device.ID)
				So(err, ShouldBeNil)
				So(*documents[0].CurrentPage, ShouldEqual, 42)
				So(*documents[0].PageCount, ShouldEqual, 171)
			})
			Convey("And the bookmark is then changed in the app", func() {
				_, err := opened.UpdateBook(book.ID, models.Book{CurrentPage: pages(200)})
				So(err, ShouldBeNil)
				Convey("Then syncing the same tablet position again leaves the app's bookmark", func() {
					book := sync()
					So(*book.CurrentPage, ShouldEqual, 200)
					So(book.ProgressSource, ShouldEqual, models.ProgressSourceApp)
				})
				Convey("Then a later tablet position replaces it", func() {
					tablet.documents = []models.RemarkableDocument{openedDocument("Dune", 100, 171, time.Now().UTC().Add(time.Minute))}
					book := sync()
					So(*book.CurrentPage, ShouldEqual, 318)
					So(book.ProgressSource, ShouldEqual, models.ProgressSourceRemarkable)
				})
			})
		})
		Convey("When a book of unknown length is open on the tablet", func() {
			_, err := opened.CreateBook(models.Book{Title: "Meeting notes"})
			So(err, ShouldBeNil)
			tablet.documents = []models.RemarkableDocument{openedDocument("Meeting notes", 12, 40, tabletTime)}
			book := sync()
			Convey("Then it takes the tablet's pages as its own", func() {
				So(*book.CurrentPage, ShouldEqual, 12)
				So(*book.PageCount, ShouldEqual, 40)
			})
		})
		Convey("When the tablet is on a book's last page", func() {
			_, err := opened.CreateBook(models.Book{Title: "Dune", PageCount: pages(544)})
			So(err, ShouldBeNil)
			tablet.documents = []models.RemarkableDocument{openedDocument("Dune", 171, 171, tabletTime)}
			book := sync()
			Convey("Then the book is marked Finished on its last page", func() {
				So(*book.CurrentPage, ShouldEqual, 544)
				So(book.Status, ShouldEqual, models.StatusFinished)
			})
		})
		Convey("When a Finished book is opened on the tablet partway through", func() {
			_, err := opened.CreateBook(models.Book{Title: "Dune", Status: models.StatusFinished, PageCount: pages(544)})
			So(err, ShouldBeNil)
			tablet.documents = []models.RemarkableDocument{openedDocument("Dune", 10, 171, tabletTime)}
			book := sync()
			Convey("Then it stays Finished with no bookmark moved", func() {
				So(book.Status, ShouldEqual, models.StatusFinished)
				So(book.CurrentPage, ShouldBeNil)
			})
		})
		Convey("When a linked document has never been opened on the tablet", func() {
			_, err := opened.CreateBook(models.Book{Title: "Dune", PageCount: pages(544)})
			So(err, ShouldBeNil)
			tablet.documents = []models.RemarkableDocument{{UUID: "document-1", Title: "Dune", FileType: models.FileTypePDF, LastModified: tabletTime}}
			book := sync()
			Convey("Then the book is left as it was", func() {
				So(book.CurrentPage, ShouldBeNil)
				So(book.Status, ShouldEqual, models.StatusWantToRead)
			})
		})
	})
}
