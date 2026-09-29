package bookfilestore_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookfilestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/storetest"
)

// epubContents starts like every EPUB: a zip whose first entry is the
// uncompressed mimetype file.
const epubContents = "PK\x03\x04" + "\x14\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x08\x00\x00\x00" + "mimetypeapplication/epub+zip" + "rest of the book"

const pdfContents = "%PDF-1.7\nrest of the book"

func TestDetectFormat(t *testing.T) {
	Convey("Given the start of files", t, func() {
		Convey("When it's an EPUB", func() {
			format, err := bookfilestore.DetectFormat([]byte(epubContents))
			Convey("Then it's detected as an EPUB", func() {
				So(err, ShouldBeNil)
				So(format, ShouldEqual, models.FileTypeEPUB)
			})
		})
		Convey("When it's a PDF", func() {
			format, err := bookfilestore.DetectFormat([]byte(pdfContents))
			Convey("Then it's detected as a PDF", func() {
				So(err, ShouldBeNil)
				So(format, ShouldEqual, models.FileTypePDF)
			})
		})
		Convey("When it's a zip that isn't an EPUB", func() {
			_, err := bookfilestore.DetectFormat([]byte("PK\x03\x04 some other archive with enough bytes to check it all"))
			Convey("Then it's unsupported", func() {
				So(errors.Is(err, bookfilestore.ErrUnsupportedFormat), ShouldBeTrue)
			})
		})
		Convey("When it's too short to tell", func() {
			_, err := bookfilestore.DetectFormat([]byte("PK"))
			Convey("Then it's unsupported", func() {
				So(errors.Is(err, bookfilestore.ErrUnsupportedFormat), ShouldBeTrue)
			})
		})
	})
}

func TestStorageServiceStore(t *testing.T) {
	Convey("Given a book on the shelf and a device", t, func() {
		stores := storetest.OpenStores(t)
		ctx := context.Background()
		book, err := stores.BookStore.Create(models.Book{Title: "Pride and Prejudice"})
		So(err, ShouldBeNil)
		device, err := stores.DeviceStore.Create(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		Convey("When saving a file that isn't an EPUB or PDF", func() {
			_, err := stores.BookFileStore.Save(ctx, bookfilestore.SaveInput{BookID: book.ID, Source: models.BookFileSourceUpload, Body: strings.NewReader("just some text")})
			Convey("Then it's refused without reaching the storage service", func() {
				So(errors.Is(err, bookfilestore.ErrUnsupportedFormat), ShouldBeTrue)
				So(stores.StorageService.Files, ShouldBeEmpty)
			})
		})
		Convey("When saving an EPUB", func() {
			saved, err := stores.BookFileStore.Save(ctx, bookfilestore.SaveInput{BookID: book.ID, Source: models.BookFileSourceOpenLibrary, Body: strings.NewReader(epubContents)})
			So(err, ShouldBeNil)
			Convey("Then it's recorded with its format, size and source", func() {
				So(saved.BookID, ShouldEqual, book.ID)
				So(saved.Format, ShouldEqual, models.FileTypeEPUB)
				So(saved.Size, ShouldEqual, len(epubContents))
				So(saved.Source, ShouldEqual, models.BookFileSourceOpenLibrary)
				So(saved.Deliveries, ShouldBeEmpty)
			})
			Convey("Then opening it returns its contents", func() {
				_, body, err := stores.BookFileStore.Open(ctx, book.ID)
				So(err, ShouldBeNil)
				defer body.Close()
				contents, _ := io.ReadAll(body)
				So(string(contents), ShouldEqual, epubContents)
			})
			Convey("Then it waits to be copied to the device", func() {
				undelivered, err := stores.BookFileStore.ListUndelivered(device.ID)
				So(err, ShouldBeNil)
				So(undelivered, ShouldHaveLength, 1)
				So(undelivered[0].BookID, ShouldEqual, book.ID)
			})
			Convey("And replacing it with a PDF", func() {
				replaced, err := stores.BookFileStore.Save(ctx, bookfilestore.SaveInput{BookID: book.ID, Source: models.BookFileSourceUpload, Body: strings.NewReader(pdfContents)})
				Convey("Then the PDF replaces it, and the EPUB leaves the storage service", func() {
					So(err, ShouldBeNil)
					So(replaced.Format, ShouldEqual, models.FileTypePDF)
					So(stores.StorageService.Files, ShouldHaveLength, 1)
				})
			})
			Convey("And recording it copied to the device", func() {
				copiedAt := time.Now().UTC()
				So(stores.BookFileStore.RecordDelivery(bookfilestore.RecordDeliveryInput{BookID: book.ID, DeviceID: device.ID, DocumentUUID: "document-1", CopiedAt: copiedAt}), ShouldBeNil)
				Convey("Then it no longer waits to be copied, but waits to be loaded", func() {
					undelivered, err := stores.BookFileStore.ListUndelivered(device.ID)
					So(err, ShouldBeNil)
					So(undelivered, ShouldBeEmpty)
					unloaded, err := stores.BookFileStore.ListUnloaded(device.ID)
					So(err, ShouldBeNil)
					So(unloaded, ShouldHaveLength, 1)
					So(unloaded[0].DocumentUUID, ShouldEqual, "document-1")
				})
				Convey("Then marking the device loaded stamps the delivery", func() {
					So(stores.BookFileStore.MarkLoaded(device.ID, copiedAt.Add(time.Minute)), ShouldBeNil)
					unloaded, err := stores.BookFileStore.ListUnloaded(device.ID)
					So(err, ShouldBeNil)
					So(unloaded, ShouldBeEmpty)
					file, err := stores.BookFileStore.Get(book.ID)
					So(err, ShouldBeNil)
					So(file.Deliveries, ShouldHaveLength, 1)
					So(file.Deliveries[0].LoadedAt, ShouldNotBeNil)
				})
				Convey("Then replacing the file keeps the delivery, so it isn't copied again", func() {
					_, err := stores.BookFileStore.Save(ctx, bookfilestore.SaveInput{BookID: book.ID, Source: models.BookFileSourceUpload, Body: strings.NewReader(pdfContents)})
					So(err, ShouldBeNil)
					undelivered, err := stores.BookFileStore.ListUndelivered(device.ID)
					So(err, ShouldBeNil)
					So(undelivered, ShouldBeEmpty)
				})
			})
			Convey("And deleting it", func() {
				So(stores.BookFileStore.Delete(ctx, book.ID), ShouldBeNil)
				Convey("Then it's gone from the store and the storage service", func() {
					_, err := stores.BookFileStore.Get(book.ID)
					So(errors.Is(err, bookfilestore.ErrFileNotFound), ShouldBeTrue)
					So(stores.StorageService.Files, ShouldBeEmpty)
				})
			})
		})
		Convey("When deleting a file the book doesn't have", func() {
			err := stores.BookFileStore.Delete(ctx, book.ID)
			Convey("Then it returns ErrFileNotFound", func() {
				So(errors.Is(err, bookfilestore.ErrFileNotFound), ShouldBeTrue)
			})
		})
	})
}
