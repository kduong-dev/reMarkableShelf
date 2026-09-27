package devicesync_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ansel1/merry"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicesync"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/remarkable"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/store"
)

// fakeTablet serves whatever documents and covers the test has put on it,
// failing with listErr or pairErr when set, and records cover requests.
type fakeTablet struct {
	covers        map[string][]byte
	coverRequests []remarkable.CoverRequest
	documents     []models.RemarkableDocument
	listErr       error
	pairErr       error
	pairedWith    string
}

func (tablet *fakeTablet) ListDocuments(host string) ([]models.RemarkableDocument, error) {
	return tablet.documents, tablet.listErr
}

func (tablet *fakeTablet) CoverImages(host string, requests []remarkable.CoverRequest) (map[string][]byte, error) {
	tablet.coverRequests = append(tablet.coverRequests, requests...)
	images := make(map[string][]byte)
	for _, request := range requests {
		if image, ok := tablet.covers[request.DocumentUUID]; ok {
			images[request.DocumentUUID] = image
		}
	}
	return images, nil
}

func (tablet *fakeTablet) Pair(host, password string) error {
	if tablet.pairErr != nil {
		return tablet.pairErr
	}
	tablet.pairedWith = password
	return nil
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

func TestLinkDocument(t *testing.T) {
	Convey("Given a synced tablet document open on page 42 of 171", t, func() {
		opened, err := store.Open(filepath.Join(t.TempDir(), "books.db"))
		So(err, ShouldBeNil)
		Reset(func() { _ = opened.Close() })
		device, err := opened.CreateDevice(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		tablet := &fakeTablet{documents: []models.RemarkableDocument{
			openedDocument("dale-carnegie.pdf", 42, 171, time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)),
		}}
		syncer := devicesync.NewSyncer(opened, tablet)
		_, err = syncer.SyncDevice(device.ID)
		So(err, ShouldBeNil)
		Convey("When linking it to a 280-page book", func() {
			book, err := opened.CreateBook(models.Book{Title: "How to Win Friends and Influence People", PageCount: pages(280)})
			So(err, ShouldBeNil)
			So(syncer.LinkDocument(device.ID, "document-1", book.ID), ShouldBeNil)
			Convey("Then the book takes the tablet's position straight away", func() {
				linked, err := opened.GetBook(book.ID)
				So(err, ShouldBeNil)
				So(*linked.CurrentPage, ShouldEqual, 69)
				So(linked.Status, ShouldEqual, models.StatusReading)
			})
		})
	})
}

func TestUnlinkDocument(t *testing.T) {
	Convey("Given a tablet document auto-linked to the book of the same title", t, func() {
		opened, err := store.Open(filepath.Join(t.TempDir(), "books.db"))
		So(err, ShouldBeNil)
		Reset(func() { _ = opened.Close() })
		device, err := opened.CreateDevice(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		book, err := opened.CreateBook(models.Book{Title: "Dune"})
		So(err, ShouldBeNil)
		tablet := &fakeTablet{documents: []models.RemarkableDocument{{UUID: "document-1", Title: "Dune", FileType: models.FileTypePDF, LastModified: time.Now().UTC()}}}
		syncer := devicesync.NewSyncer(opened, tablet)
		linkedTo := func() *string {
			documents, err := syncer.SyncDevice(device.ID)
			So(err, ShouldBeNil)
			return documents[0].LinkedBookID
		}
		So(*linkedTo(), ShouldEqual, book.ID)
		Convey("When the user unlinks it and the tablet syncs again", func() {
			So(opened.UnlinkDocument(device.ID, "document-1"), ShouldBeNil)
			Convey("Then sync doesn't link it again by title", func() {
				So(linkedTo(), ShouldBeNil)
			})
			Convey("Then the user can still link it by hand", func() {
				So(syncer.LinkDocument(device.ID, "document-1", book.ID), ShouldBeNil)
				So(*linkedTo(), ShouldEqual, book.ID)
			})
		})
	})
}

func TestCovers(t *testing.T) {
	Convey("Given a tablet with a PDF and a notebook whose covers are rendered", t, func() {
		opened, err := store.Open(filepath.Join(t.TempDir(), "books.db"))
		So(err, ShouldBeNil)
		Reset(func() { _ = opened.Close() })
		device, err := opened.CreateDevice(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		modifiedAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
		pdf := models.RemarkableDocument{UUID: "document-1", Title: "Dune", FileType: models.FileTypePDF, LastModified: modifiedAt, CoverPageID: "page-1"}
		notebook := models.RemarkableDocument{UUID: "notebook-1", Title: "Notes", FileType: models.FileTypeNotebook, LastModified: modifiedAt, CoverPageID: "page-2"}
		tablet := &fakeTablet{
			documents: []models.RemarkableDocument{pdf, notebook},
			covers:    map[string][]byte{"document-1": []byte("dune cover"), "notebook-1": []byte("notes cover")},
		}
		syncer := devicesync.NewSyncer(opened, tablet)
		book, err := opened.CreateBook(models.Book{Title: "Dune"})
		So(err, ShouldBeNil)
		_, err = syncer.SyncDevice(device.ID)
		So(err, ShouldBeNil)
		Convey("When the first sync runs", func() {
			Convey("Then only the PDF's cover is fetched and stored", func() {
				So(tablet.coverRequests, ShouldResemble, []remarkable.CoverRequest{{DocumentUUID: "document-1", PageID: "page-1"}})
				image, err := opened.GetDocumentCover(device.ID, "document-1")
				So(err, ShouldBeNil)
				So(string(image), ShouldEqual, "dune cover")
			})
			Convey("Then the book it auto-linked to offers the tablet's cover", func() {
				linked, err := opened.GetBook(book.ID)
				So(err, ShouldBeNil)
				So(linked.TabletCoverURL, ShouldEqual, "/api/devices/"+device.ID+"/documents/document-1/cover")
			})
		})
		Convey("When syncing again with the document unchanged", func() {
			tablet.coverRequests = nil
			_, err := syncer.SyncDevice(device.ID)
			Convey("Then the cover isn't fetched again", func() {
				So(err, ShouldBeNil)
				So(tablet.coverRequests, ShouldBeEmpty)
			})
		})
		Convey("When the document changes and its cover with it", func() {
			tablet.coverRequests = nil
			pdf.LastModified = modifiedAt.Add(time.Minute)
			tablet.documents = []models.RemarkableDocument{pdf}
			tablet.covers["document-1"] = []byte("new cover")
			_, err := syncer.SyncDevice(device.ID)
			Convey("Then the new cover replaces the old", func() {
				So(err, ShouldBeNil)
				So(tablet.coverRequests, ShouldHaveLength, 1)
				image, err := opened.GetDocumentCover(device.ID, "document-1")
				So(err, ShouldBeNil)
				So(string(image), ShouldEqual, "new cover")
			})
		})
	})
	Convey("Given a PDF whose cover the tablet hasn't rendered yet", t, func() {
		opened, err := store.Open(filepath.Join(t.TempDir(), "books.db"))
		So(err, ShouldBeNil)
		Reset(func() { _ = opened.Close() })
		device, err := opened.CreateDevice(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		tablet := &fakeTablet{documents: []models.RemarkableDocument{
			{UUID: "document-1", Title: "Dune", FileType: models.FileTypePDF, LastModified: time.Now().UTC(), CoverPageID: "page-1"},
		}}
		syncer := devicesync.NewSyncer(opened, tablet)
		_, err = syncer.SyncDevice(device.ID)
		So(err, ShouldBeNil)
		Convey("When it's rendered before the next sync", func() {
			tablet.covers = map[string][]byte{"document-1": []byte("dune cover")}
			_, err := syncer.SyncDevice(device.ID)
			Convey("Then that sync picks it up", func() {
				So(err, ShouldBeNil)
				image, err := opened.GetDocumentCover(device.ID, "document-1")
				So(err, ShouldBeNil)
				So(string(image), ShouldEqual, "dune cover")
			})
		})
	})
}

func TestPairing(t *testing.T) {
	Convey("Given a syncer with no devices", t, func() {
		opened, err := store.Open(filepath.Join(t.TempDir(), "books.db"))
		So(err, ShouldBeNil)
		Reset(func() { _ = opened.Close() })
		tablet := &fakeTablet{}
		syncer := devicesync.NewSyncer(opened, tablet)
		register := func() (models.Device, error) {
			return syncer.RegisterDevice(devicesync.RegisterDeviceInput{Name: " Paper Pro ", Host: "10.0.0.5", Password: "secret"})
		}
		Convey("When registering a tablet with its password", func() {
			device, err := register()
			Convey("Then it pairs with the password and saves the device as paired", func() {
				So(err, ShouldBeNil)
				So(tablet.pairedWith, ShouldEqual, "secret")
				So(device.Name, ShouldEqual, "Paper Pro")
				stored, err := opened.GetDevice(device.ID)
				So(err, ShouldBeNil)
				So(stored.PairedAt, ShouldNotBeNil)
			})
			Convey("And the tablet later rejects the server's key", func() {
				tablet.listErr = merry.Here(remarkable.ErrNotPaired)
				_, err := syncer.SyncDevice(device.ID)
				Convey("Then the sync fails and the device is marked unpaired", func() {
					So(merry.Is(err, remarkable.ErrNotPaired), ShouldBeTrue)
					stored, err := opened.GetDevice(device.ID)
					So(err, ShouldBeNil)
					So(stored.PairedAt, ShouldBeNil)
				})
				Convey("Then pairing it again marks it paired", func() {
					tablet.listErr = nil
					repaired, err := syncer.PairDevice(device.ID, "secret")
					So(err, ShouldBeNil)
					So(repaired.PairedAt, ShouldNotBeNil)
				})
			})
			Convey("And the tablet is just asleep", func() {
				tablet.listErr = merry.Here(remarkable.ErrTabletUnreachable)
				_, err := syncer.SyncDevice(device.ID)
				Convey("Then the device stays paired", func() {
					So(merry.Is(err, remarkable.ErrTabletUnreachable), ShouldBeTrue)
					stored, err := opened.GetDevice(device.ID)
					So(err, ShouldBeNil)
					So(stored.PairedAt, ShouldNotBeNil)
				})
			})
		})
		Convey("When registering with a password the tablet rejects", func() {
			tablet.pairErr = merry.Here(remarkable.ErrWrongPassword)
			_, err := register()
			Convey("Then it returns ErrWrongPassword and saves no device", func() {
				So(merry.Is(err, remarkable.ErrWrongPassword), ShouldBeTrue)
				devices, err := opened.ListDevices()
				So(err, ShouldBeNil)
				So(devices, ShouldBeEmpty)
			})
		})
		Convey("When registering without a password", func() {
			_, err := syncer.RegisterDevice(devicesync.RegisterDeviceInput{Name: "Paper Pro", Host: "10.0.0.5"})
			Convey("Then it returns ErrPasswordRequired without contacting the tablet", func() {
				So(merry.Is(err, devicesync.ErrPasswordRequired), ShouldBeTrue)
				So(tablet.pairedWith, ShouldBeEmpty)
			})
		})
		Convey("When registering without a host", func() {
			_, err := syncer.RegisterDevice(devicesync.RegisterDeviceInput{Name: "Paper Pro", Password: "secret"})
			Convey("Then it returns ErrDeviceDetailsRequired", func() {
				So(merry.Is(err, devicesync.ErrDeviceDetailsRequired), ShouldBeTrue)
			})
		})
	})
}
