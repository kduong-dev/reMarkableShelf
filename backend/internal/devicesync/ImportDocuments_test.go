package devicesync_test

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicesync"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/storetest"
)

func TestSyncImportsDocuments(t *testing.T) {
	Convey("Given a tablet with books no book in the library matches, and other documents", t, func() {
		stores := storetest.OpenStores(t)
		device, err := stores.DeviceStore.Create(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		_, err = stores.BookStore.Create(models.Book{Title: "Dune"})
		So(err, ShouldBeNil)
		openedAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
		tablet := &fakeTablet{
			folders: []models.RemarkableFolder{{UUID: "old", Title: "Old", ParentUUID: models.TrashFolderUUID}},
			documents: []models.RemarkableDocument{
				{UUID: "dune", Title: "Dune", FileType: models.FileTypePDF},
				{UUID: "carnegie", Title: "Dale Carnegie - How to Win Friends (Veridian Digital Press).pdf", FileType: models.FileTypePDF,
					BookTitle: "Microsoft Word - carnegie.docx", CurrentPage: pages(42), PageCount: pages(171), PositionUpdatedAt: &openedAt, LastModified: openedAt},
				{UUID: "crucial", Title: "Crucial Conversations_ Tools for Talking (2011, McGraw-Hill).pdf", FileType: models.FileTypePDF},
				{UUID: "nations", Title: "Why Nations Fail", FileType: models.FileTypeEPUB, BookTitle: "Why Nations Fail: The Origins of Power", BookAuthor: "Daron Acemoglu"},
				{UUID: "untitled", Title: "The Time Machine", FileType: models.FileTypeEPUB},
				{UUID: "sketch", Title: "Scribbles", FileType: models.FileTypeNotebook},
				{UUID: "journal", Title: "Journal 2025", FileType: models.FileTypePDF},
				{UUID: "binned", Title: "Draft.pdf", FileType: models.FileTypePDF, ParentUUID: models.TrashFolderUUID},
				{UUID: "deep", Title: "Old notes.pdf", FileType: models.FileTypePDF, ParentUUID: "old"},
			},
		}
		So(stores.DocumentStore.Upsert(device.ID, tablet.documents), ShouldBeNil)
		So(stores.DocumentStore.SetNotABook(device.ID, "journal", true), ShouldBeNil)
		syncer := devicesync.NewSyncer(devicesync.NewSyncerInput{
			BookFileStore: stores.BookFileStore,
			BookStore:     stores.BookStore,
			DeviceStore:   stores.DeviceStore,
			DocumentStore: stores.DocumentStore,
			Remarkable:    tablet,
		})
		booksByDocument := func() map[string]models.Book {
			documents, err := stores.DocumentStore.ListByDevice(device.ID)
			So(err, ShouldBeNil)
			books := map[string]models.Book{}
			for _, document := range documents {
				if document.LinkedBookID != nil {
					book, err := stores.BookStore.Get(*document.LinkedBookID)
					So(err, ShouldBeNil)
					books[document.UUID] = book
				}
			}
			return books
		}
		Convey("When syncing", func() {
			_, err := syncer.SyncDevice(device.ID)
			So(err, ShouldBeNil)
			books := booksByDocument()
			Convey("Then the matching book is linked by title rather than imported", func() {
				library, err := stores.BookStore.List()
				So(err, ShouldBeNil)
				So(library, ShouldHaveLength, 5)
				So(books["dune"].Source, ShouldEqual, models.SourceManual)
			})
			Convey("Then each other PDF or EPUB is added as is, from the tablet", func() {
				So(books["carnegie"].Source, ShouldEqual, models.SourceRemarkable)
				So(books["carnegie"].OpenLibraryID, ShouldBeEmpty)
			})
			Convey("Then a PDF is named after its tidied file name, not its embedded title", func() {
				So(books["carnegie"].Title, ShouldEqual, "Dale Carnegie - How to Win Friends")
				So(books["crucial"].Title, ShouldEqual, "Crucial Conversations: Tools for Talking")
			})
			Convey("Then an EPUB is named as the tablet shows it, with the author from the file", func() {
				So(books["nations"].Title, ShouldEqual, "Why Nations Fail")
				So(books["nations"].Author, ShouldEqual, "Daron Acemoglu")
				So(books["untitled"].Title, ShouldEqual, "The Time Machine")
			})
			Convey("Then an imported book takes the tablet's reading position", func() {
				So(*books["carnegie"].CurrentPage, ShouldEqual, 42)
				So(books["carnegie"].Status, ShouldEqual, models.StatusReading)
			})
			Convey("Then notebooks, documents marked not a book and ones in the trash are left out", func() {
				for _, uuid := range []string{"sketch", "journal", "binned", "deep"} {
					So(books, ShouldNotContainKey, uuid)
				}
			})
			Convey("And syncing again", func() {
				_, err := syncer.SyncDevice(device.ID)
				So(err, ShouldBeNil)
				Convey("Then nothing is imported twice", func() {
					library, err := stores.BookStore.List()
					So(err, ShouldBeNil)
					So(library, ShouldHaveLength, 5)
				})
			})
			Convey("And an imported book is deleted from the library before syncing again", func() {
				So(stores.DocumentStore.UnlinkBook(books["nations"].ID), ShouldBeNil)
				So(stores.BookStore.Delete(books["nations"].ID), ShouldBeNil)
				_, err := syncer.SyncDevice(device.ID)
				So(err, ShouldBeNil)
				Convey("Then it isn't imported again", func() {
					So(booksByDocument(), ShouldNotContainKey, "nations")
					library, err := stores.BookStore.List()
					So(err, ShouldBeNil)
					So(library, ShouldHaveLength, 4)
				})
			})
		})
	})
}
