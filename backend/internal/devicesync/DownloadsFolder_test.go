package devicesync_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookfilestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicesync"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/storetest"
)

func TestSyncCopiesIntoDownloadsFolder(t *testing.T) {
	Convey("Given a paired tablet and two books with files saved on the server", t, func() {
		stores := storetest.OpenStores(t)
		device, err := stores.DeviceStore.Create(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		saveBook := func(title string) models.Book {
			book, err := stores.BookStore.Create(models.Book{Title: title})
			So(err, ShouldBeNil)
			_, err = stores.BookFileStore.Save(context.Background(), bookfilestore.SaveInput{
				BookID: book.ID, Source: models.BookFileSourceUpload, Body: strings.NewReader("%PDF-1.7 " + title),
			})
			So(err, ShouldBeNil)
			return book
		}
		saveBook("Emma")
		saveBook("Persuasion")
		tablet := &fakeTablet{}
		syncer := devicesync.NewSyncer(devicesync.NewSyncerInput{
			BookFileStore: stores.BookFileStore,
			BookStore:     stores.BookStore,
			DeviceStore:   stores.DeviceStore,
			DocumentStore: stores.DocumentStore,
			Remarkable:    tablet,
		})
		sync := func() []models.RemarkableDocument {
			documents, err := syncer.SyncDevice(device.ID)
			So(err, ShouldBeNil)
			return documents
		}
		Convey("When syncing a tablet with no Downloads folder", func() {
			documents := sync()
			Convey("Then one Downloads folder is created at the top of the library", func() {
				So(tablet.createdFolders, ShouldHaveLength, 1)
				So(tablet.createdFolders[0].Title, ShouldEqual, devicesync.DownloadsFolderTitle)
				So(tablet.createdFolders[0].ParentUUID, ShouldEqual, "")
			})
			Convey("Then both books are copied into it", func() {
				folderUUID := tablet.createdFolders[0].UUID
				So(tablet.copied, ShouldHaveLength, 2)
				for _, copied := range tablet.copied {
					So(copied.input.ParentUUID, ShouldEqual, folderUUID)
				}
				for _, document := range documents {
					So(document.ParentUUID, ShouldEqual, folderUUID)
				}
			})
			Convey("Then the folder shows among the synced folders", func() {
				folders, err := stores.DocumentStore.ListFolders(device.ID)
				So(err, ShouldBeNil)
				So(folders, ShouldResemble, tablet.createdFolders)
			})
			Convey("And a third book is saved and the Downloads folder renamed on the tablet", func() {
				tablet.folders[0].Title = "From the shelf"
				saveBook("Sanditon")
				sync()
				Convey("Then the third book goes in the same folder, without creating another", func() {
					So(tablet.createdFolders, ShouldHaveLength, 1)
					So(tablet.copied, ShouldHaveLength, 3)
					So(tablet.copied[2].input.ParentUUID, ShouldEqual, tablet.createdFolders[0].UUID)
				})
			})
			Convey("And the Downloads folder is trashed before a third book is saved", func() {
				tablet.folders[0].ParentUUID = models.TrashFolderUUID
				saveBook("Sanditon")
				sync()
				Convey("Then a new Downloads folder is created for it", func() {
					So(tablet.createdFolders, ShouldHaveLength, 2)
					So(tablet.copied[2].input.ParentUUID, ShouldEqual, tablet.createdFolders[1].UUID)
				})
			})
		})
		Convey("When syncing a tablet that already has a Downloads folder", func() {
			tablet.folders = []models.RemarkableFolder{{UUID: "existing", Title: "downloads"}}
			sync()
			Convey("Then the books are copied into it without creating another", func() {
				So(tablet.createdFolders, ShouldBeEmpty)
				So(tablet.copied[0].input.ParentUUID, ShouldEqual, "existing")
			})
		})
		Convey("When the tablet can't be written to", func() {
			tablet.copyErr = errors.New("connection reset")
			_, err := syncer.SyncDevice(device.ID)
			Convey("Then the sync still succeeds, and the books wait for the next sync", func() {
				So(err, ShouldBeNil)
				So(tablet.copied, ShouldBeEmpty)
				undelivered, err := stores.BookFileStore.ListUndelivered(device.ID)
				So(err, ShouldBeNil)
				So(undelivered, ShouldHaveLength, 2)
			})
		})
		Convey("When syncing with nothing to copy", func() {
			fresh := storetest.OpenStores(t)
			quietDevice, err := fresh.DeviceStore.Create(models.Device{Name: "Paper Pro", Host: "10.0.0.6"})
			So(err, ShouldBeNil)
			quietTablet := &fakeTablet{}
			_, err = devicesync.NewSyncer(devicesync.NewSyncerInput{
				BookFileStore: fresh.BookFileStore,
				BookStore:     fresh.BookStore,
				DeviceStore:   fresh.DeviceStore,
				DocumentStore: fresh.DocumentStore,
				Remarkable:    quietTablet,
			}).SyncDevice(quietDevice.ID)
			Convey("Then no folder is created", func() {
				So(err, ShouldBeNil)
				So(quietTablet.createdFolders, ShouldBeEmpty)
			})
		})
	})
}
