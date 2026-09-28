package store_test

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/store"
)

func TestForeignKeys(t *testing.T) {
	Convey("Given a book linked to a document on a device", t, func() {
		opened, err := store.Open(filepath.Join(t.TempDir(), "books.db"))
		So(err, ShouldBeNil)
		Reset(func() { _ = opened.Close() })
		device, err := opened.Devices.Create(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		So(opened.Documents.Upsert(device.ID, []models.RemarkableDocument{{UUID: "document-1", Title: "Dune.pdf", FileType: models.FileTypePDF, LastModified: time.Now().UTC()}}), ShouldBeNil)
		book, err := opened.Books.Create(models.Book{Title: "Dune"})
		So(err, ShouldBeNil)
		So(opened.Documents.LinkToBook(device.ID, "document-1", book.ID), ShouldBeNil)
		Convey("When the book is deleted", func() {
			So(opened.Books.Delete(book.ID), ShouldBeNil)
			Convey("Then the document is unlinked rather than pointing at a missing book", func() {
				documents, err := opened.Documents.ListByDevice(device.ID)
				So(err, ShouldBeNil)
				So(documents[0].LinkedBookID, ShouldBeNil)
			})
		})
		Convey("When the device is deleted", func() {
			So(opened.Devices.Delete(device.ID), ShouldBeNil)
			Convey("Then its documents go with it and the book stays", func() {
				documents, err := opened.Documents.ListByDevice(device.ID)
				So(err, ShouldBeNil)
				So(documents, ShouldBeEmpty)
				_, err = opened.Books.Get(book.ID)
				So(err, ShouldBeNil)
			})
		})
	})
	Convey("Given a database written while foreign keys weren't enforced", t, func() {
		path := filepath.Join(t.TempDir(), "books.db")
		opened, err := store.Open(path)
		So(err, ShouldBeNil)
		device, err := opened.Devices.Create(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		So(opened.Documents.Upsert(device.ID, []models.RemarkableDocument{{UUID: "document-1", Title: "Dune.pdf", FileType: models.FileTypePDF, LastModified: time.Now().UTC()}}), ShouldBeNil)
		So(opened.Close(), ShouldBeNil)
		unenforced, err := sql.Open("sqlite", path)
		So(err, ShouldBeNil)
		_, err = unenforced.Exec(`UPDATE remarkable_documents SET linked_book_id = 'deleted-book';
			INSERT INTO remarkable_documents (uuid, device_id, title, file_type) VALUES ('orphan', 'deleted-device', 'x.pdf', 'pdf')`)
		So(err, ShouldBeNil)
		So(unenforced.Close(), ShouldBeNil)
		Convey("When the store opens it", func() {
			reopened, err := store.Open(path)
			So(err, ShouldBeNil)
			Reset(func() { _ = reopened.Close() })
			Convey("Then the link to the deleted book is cleared", func() {
				documents, err := reopened.Documents.ListByDevice(device.ID)
				So(err, ShouldBeNil)
				So(documents[0].LinkedBookID, ShouldBeNil)
			})
			Convey("Then documents of the deleted device are removed", func() {
				documents, err := reopened.Documents.ListByDevice("deleted-device")
				So(err, ShouldBeNil)
				So(documents, ShouldBeEmpty)
			})
		})
	})
}
