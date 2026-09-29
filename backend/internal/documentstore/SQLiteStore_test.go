package documentstore_test

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/storetest"
)

func TestUpsert(t *testing.T) {
	Convey("Given a device with two synced documents, one linked to a book", t, func() {
		bookStore, deviceStore, documentStore := storetest.Open(t)
		device, err := deviceStore.Create(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		book, err := bookStore.Create(models.Book{Title: "Dune"})
		So(err, ShouldBeNil)
		now := time.Now().UTC()
		dune := models.RemarkableDocument{UUID: "dune", Title: "Dune", FileType: models.FileTypeEPUB, LastModified: now, ParentUUID: "books"}
		notes := models.RemarkableDocument{UUID: "notes", Title: "Notes", FileType: models.FileTypeNotebook, LastModified: now}
		So(documentStore.Upsert(device.ID, []models.RemarkableDocument{dune, notes}), ShouldBeNil)
		So(documentStore.LinkToBook(device.ID, "dune", book.ID), ShouldBeNil)
		Convey("When listing them", func() {
			documents, err := documentStore.ListByDevice(device.ID)
			Convey("Then each keeps its folder", func() {
				So(err, ShouldBeNil)
				parents := map[string]string{}
				for _, document := range documents {
					parents[document.UUID] = document.ParentUUID
				}
				So(parents, ShouldResemble, map[string]string{"dune": "books", "notes": ""})
			})
		})
		Convey("When a sync no longer finds the linked document", func() {
			So(documentStore.Upsert(device.ID, []models.RemarkableDocument{notes}), ShouldBeNil)
			Convey("Then it's left out of the device's and the book's documents", func() {
				documents, err := documentStore.ListByDevice(device.ID)
				So(err, ShouldBeNil)
				So(documents, ShouldHaveLength, 1)
				So(documents[0].UUID, ShouldEqual, "notes")
				linked, err := documentStore.ListByBook(book.ID)
				So(err, ShouldBeNil)
				So(linked, ShouldBeEmpty)
			})
			Convey("Then the book no longer takes its page count or cover", func() {
				stored, err := bookStore.Get(book.ID)
				So(err, ShouldBeNil)
				So(stored.TabletPageCount, ShouldBeNil)
			})
			Convey("And a later sync finds it again, moved to another folder", func() {
				dune.ParentUUID = ""
				So(documentStore.Upsert(device.ID, []models.RemarkableDocument{dune, notes}), ShouldBeNil)
				Convey("Then it's back, still linked, in its new folder", func() {
					linked, err := documentStore.ListByBook(book.ID)
					So(err, ShouldBeNil)
					So(linked, ShouldHaveLength, 1)
					So(linked[0].ParentUUID, ShouldEqual, "")
				})
			})
		})
		Convey("When a sync finds no documents", func() {
			So(documentStore.Upsert(device.ID, nil), ShouldBeNil)
			Convey("Then every document is removed", func() {
				documents, err := documentStore.ListByDevice(device.ID)
				So(err, ShouldBeNil)
				So(documents, ShouldBeEmpty)
			})
		})
	})
}

func TestReplaceFolders(t *testing.T) {
	Convey("Given a device with synced folders", t, func() {
		_, deviceStore, documentStore := storetest.Open(t)
		device, err := deviceStore.Create(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		So(documentStore.ReplaceFolders(device.ID, []models.RemarkableFolder{
			{UUID: "books", Title: "Books"},
			{UUID: "classics", Title: "Classics", ParentUUID: "books"},
		}), ShouldBeNil)
		Convey("When a sync finds one renamed and the other gone", func() {
			So(documentStore.ReplaceFolders(device.ID, []models.RemarkableFolder{{UUID: "books", Title: "Reading"}}), ShouldBeNil)
			Convey("Then the folders match the tablet's", func() {
				folders, err := documentStore.ListFolders(device.ID)
				So(err, ShouldBeNil)
				So(folders, ShouldResemble, []models.RemarkableFolder{{UUID: "books", Title: "Reading"}})
			})
		})
	})
}
