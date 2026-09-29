package devicesync_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookfilestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicesync"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/remarkable"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/storetest"
)

// fakeTablet serves whatever documents and covers the test has put on it,
// presenting hostKey, failing with listErr or pairErr when set, and records
// cover requests and the tablet it was last asked to reach.
type fakeTablet struct {
	hostKey       string
	lastTarget    remarkable.Tablet
	covers        map[string][]byte
	coverRequests []remarkable.CoverRequest
	documents     []models.RemarkableDocument
	listErr       error
	pairErr       error
	pairedWith    string
	copied        []copiedDocument
	copyErr       error
	restarts      int
}

type copiedDocument struct {
	input    remarkable.CopyDocumentInput
	contents string
}

func (tablet *fakeTablet) ListDocuments(target remarkable.Tablet) (remarkable.Listing, error) {
	tablet.lastTarget = target
	if tablet.listErr != nil {
		return remarkable.Listing{}, tablet.listErr
	}
	return remarkable.Listing{Documents: tablet.documents, HostKey: tablet.hostKey}, nil
}

func (tablet *fakeTablet) CoverImages(target remarkable.Tablet, requests []remarkable.CoverRequest) (map[string][]byte, error) {
	tablet.coverRequests = append(tablet.coverRequests, requests...)
	images := make(map[string][]byte)
	for _, request := range requests {
		if image, ok := tablet.covers[request.DocumentUUID]; ok {
			images[request.DocumentUUID] = image
		}
	}
	return images, nil
}

func (tablet *fakeTablet) CopyDocument(target remarkable.Tablet, input remarkable.CopyDocumentInput) error {
	if tablet.copyErr != nil {
		return tablet.copyErr
	}
	contents, err := io.ReadAll(input.Body)
	if err != nil {
		return err
	}
	tablet.copied = append(tablet.copied, copiedDocument{input: input, contents: string(contents)})
	// The copy lands in the documents directory, so the next listing finds it.
	tablet.documents = append(tablet.documents, models.RemarkableDocument{UUID: input.UUID, Title: input.Title, FileType: input.FileType, LastModified: time.Now().UTC()})
	return nil
}

func (tablet *fakeTablet) RestartApp(target remarkable.Tablet) error {
	tablet.restarts++
	return nil
}

func (tablet *fakeTablet) Pair(target remarkable.Tablet, password string) (string, error) {
	tablet.lastTarget = target
	if tablet.pairErr != nil {
		return "", tablet.pairErr
	}
	tablet.pairedWith = password
	return tablet.hostKey, nil
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
		stores := storetest.OpenStores(t)
		bookStore, deviceStore, documentStore, bookFileStore := stores.BookStore, stores.DeviceStore, stores.DocumentStore, stores.BookFileStore
		device, err := deviceStore.Create(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		tablet := &fakeTablet{}
		syncer := devicesync.NewSyncer(devicesync.NewSyncerInput{BookFileStore: bookFileStore, BookStore: bookStore, DeviceStore: deviceStore, DocumentStore: documentStore, Remarkable: tablet})
		sync := func() models.Book {
			_, err := syncer.SyncDevice(device.ID)
			So(err, ShouldBeNil)
			books, err := bookStore.List()
			So(err, ShouldBeNil)
			return books[0]
		}
		// An hour ago, at the millisecond precision the tablet records.
		tabletTime := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)

		Convey("When a 544-page book's PDF is on page 42 of 171 on the tablet", func() {
			_, err := bookStore.Create(models.Book{Title: "Dune", PageCount: pages(544)})
			So(err, ShouldBeNil)
			tablet.documents = []models.RemarkableDocument{openedDocument("Dune", 42, 171, tabletTime)}
			book := sync()
			Convey("Then the document is linked by title and the book takes the tablet's page and page count", func() {
				So(*book.CurrentPage, ShouldEqual, 42)
				So(*book.PageCount, ShouldEqual, 171)
				So(book.Status, ShouldEqual, models.StatusReading)
				So(book.ProgressSource, ShouldEqual, models.ProgressSourceRemarkable)
				So(*book.ProgressUpdatedAt, ShouldEqual, tabletTime)
			})
			Convey("Then the tablet's own position is kept on the document", func() {
				documents, err := documentStore.ListByDevice(device.ID)
				So(err, ShouldBeNil)
				So(*documents[0].CurrentPage, ShouldEqual, 42)
				So(*documents[0].PageCount, ShouldEqual, 171)
			})
			Convey("And the bookmark is then changed in the app", func() {
				_, err := bookStore.Update(book.ID, models.Book{CurrentPage: pages(60)})
				So(err, ShouldBeNil)
				Convey("Then syncing the same tablet position again leaves the app's bookmark", func() {
					book := sync()
					So(*book.CurrentPage, ShouldEqual, 60)
					So(book.ProgressSource, ShouldEqual, models.ProgressSourceApp)
				})
				Convey("Then a later tablet position replaces it", func() {
					tablet.documents = []models.RemarkableDocument{openedDocument("Dune", 100, 171, time.Now().UTC().Add(time.Minute))}
					book := sync()
					So(*book.CurrentPage, ShouldEqual, 100)
					So(book.ProgressSource, ShouldEqual, models.ProgressSourceRemarkable)
				})
			})
		})
		Convey("When a book of unknown length is open on the tablet", func() {
			_, err := bookStore.Create(models.Book{Title: "Meeting notes"})
			So(err, ShouldBeNil)
			tablet.documents = []models.RemarkableDocument{openedDocument("Meeting notes", 12, 40, tabletTime)}
			book := sync()
			Convey("Then it takes the tablet's pages as its own", func() {
				So(*book.CurrentPage, ShouldEqual, 12)
				So(*book.PageCount, ShouldEqual, 40)
			})
		})
		Convey("When the tablet is on a book's last page", func() {
			_, err := bookStore.Create(models.Book{Title: "Dune", PageCount: pages(544)})
			So(err, ShouldBeNil)
			tablet.documents = []models.RemarkableDocument{openedDocument("Dune", 171, 171, tabletTime)}
			book := sync()
			Convey("Then the book is marked Finished on its last page", func() {
				So(*book.CurrentPage, ShouldEqual, 171)
				So(book.Status, ShouldEqual, models.StatusFinished)
			})
		})
		Convey("When a bookmarked book is linked to a document with no newer position", func() {
			_, err := bookStore.Create(models.Book{Title: "Dune", PageCount: pages(544), CurrentPage: pages(272)})
			So(err, ShouldBeNil)
			tablet.documents = []models.RemarkableDocument{openedDocument("Dune", 10, 171, tabletTime)}
			book := sync()
			Convey("Then it still takes the tablet's page count, keeping the bookmark's place", func() {
				So(*book.PageCount, ShouldEqual, 171)
				So(*book.CurrentPage, ShouldEqual, 86)
				So(book.ProgressSource, ShouldEqual, models.ProgressSourceApp)
			})
		})
		Convey("When a Finished book is opened on the tablet partway through", func() {
			_, err := bookStore.Create(models.Book{Title: "Dune", Status: models.StatusFinished, PageCount: pages(544)})
			So(err, ShouldBeNil)
			tablet.documents = []models.RemarkableDocument{openedDocument("Dune", 10, 171, tabletTime)}
			book := sync()
			Convey("Then it stays Finished with no bookmark moved", func() {
				So(book.Status, ShouldEqual, models.StatusFinished)
				So(book.CurrentPage, ShouldBeNil)
			})
		})
		Convey("When a linked document has never been opened on the tablet", func() {
			_, err := bookStore.Create(models.Book{Title: "Dune", PageCount: pages(544)})
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
		stores := storetest.OpenStores(t)
		bookStore, deviceStore, documentStore, bookFileStore := stores.BookStore, stores.DeviceStore, stores.DocumentStore, stores.BookFileStore
		device, err := deviceStore.Create(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		tablet := &fakeTablet{documents: []models.RemarkableDocument{
			openedDocument("dale-carnegie.pdf", 42, 171, time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)),
		}}
		syncer := devicesync.NewSyncer(devicesync.NewSyncerInput{BookFileStore: bookFileStore, BookStore: bookStore, DeviceStore: deviceStore, DocumentStore: documentStore, Remarkable: tablet})
		_, err = syncer.SyncDevice(device.ID)
		So(err, ShouldBeNil)
		Convey("When linking it to a 280-page book", func() {
			book, err := bookStore.Create(models.Book{Title: "How to Win Friends and Influence People", PageCount: pages(280)})
			So(err, ShouldBeNil)
			So(syncer.LinkDocument(device.ID, "document-1", book.ID), ShouldBeNil)
			Convey("Then the book takes the tablet's page and page count straight away", func() {
				linked, err := bookStore.Get(book.ID)
				So(err, ShouldBeNil)
				So(*linked.CurrentPage, ShouldEqual, 42)
				So(*linked.PageCount, ShouldEqual, 171)
				So(*linked.TabletPageCount, ShouldEqual, 171)
				So(linked.Status, ShouldEqual, models.StatusReading)
			})
		})
	})
}

func TestAlignBook(t *testing.T) {
	Convey("Given a book linked to a 171-page tablet document open on page 42", t, func() {
		stores := storetest.OpenStores(t)
		bookStore, deviceStore, documentStore, bookFileStore := stores.BookStore, stores.DeviceStore, stores.DocumentStore, stores.BookFileStore
		device, err := deviceStore.Create(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		book, err := bookStore.Create(models.Book{Title: "Dune"})
		So(err, ShouldBeNil)
		tablet := &fakeTablet{documents: []models.RemarkableDocument{
			openedDocument("Dune", 42, 171, time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)),
		}}
		syncer := devicesync.NewSyncer(devicesync.NewSyncerInput{BookFileStore: bookFileStore, BookStore: bookStore, DeviceStore: deviceStore, DocumentStore: documentStore, Remarkable: tablet})
		_, err = syncer.SyncDevice(device.ID)
		So(err, ShouldBeNil)
		Convey("When an edit gives the book another edition's page count", func() {
			_, err := bookStore.Update(book.ID, models.Book{PageCount: pages(280)})
			So(err, ShouldBeNil)
			So(syncer.AlignBook(book.ID), ShouldBeNil)
			Convey("Then aligning puts it back on the tablet's pages", func() {
				aligned, err := bookStore.Get(book.ID)
				So(err, ShouldBeNil)
				So(*aligned.PageCount, ShouldEqual, 171)
				So(*aligned.CurrentPage, ShouldEqual, 42)
			})
		})
	})
}

func TestUnlinkDocument(t *testing.T) {
	Convey("Given a tablet document auto-linked to the book of the same title", t, func() {
		stores := storetest.OpenStores(t)
		bookStore, deviceStore, documentStore, bookFileStore := stores.BookStore, stores.DeviceStore, stores.DocumentStore, stores.BookFileStore
		device, err := deviceStore.Create(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		book, err := bookStore.Create(models.Book{Title: "Dune"})
		So(err, ShouldBeNil)
		tablet := &fakeTablet{documents: []models.RemarkableDocument{{UUID: "document-1", Title: "Dune", FileType: models.FileTypePDF, LastModified: time.Now().UTC()}}}
		syncer := devicesync.NewSyncer(devicesync.NewSyncerInput{BookFileStore: bookFileStore, BookStore: bookStore, DeviceStore: deviceStore, DocumentStore: documentStore, Remarkable: tablet})
		linkedTo := func() *string {
			documents, err := syncer.SyncDevice(device.ID)
			So(err, ShouldBeNil)
			return documents[0].LinkedBookID
		}
		So(*linkedTo(), ShouldEqual, book.ID)
		Convey("When the user unlinks it and the tablet syncs again", func() {
			So(documentStore.Unlink(device.ID, "document-1"), ShouldBeNil)
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
		stores := storetest.OpenStores(t)
		bookStore, deviceStore, documentStore, bookFileStore := stores.BookStore, stores.DeviceStore, stores.DocumentStore, stores.BookFileStore
		device, err := deviceStore.Create(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		modifiedAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
		pdf := models.RemarkableDocument{UUID: "document-1", Title: "Dune", FileType: models.FileTypePDF, LastModified: modifiedAt, CoverPageID: "page-1"}
		notebook := models.RemarkableDocument{UUID: "notebook-1", Title: "Notes", FileType: models.FileTypeNotebook, LastModified: modifiedAt, CoverPageID: "page-2"}
		tablet := &fakeTablet{
			documents: []models.RemarkableDocument{pdf, notebook},
			covers:    map[string][]byte{"document-1": []byte("dune cover"), "notebook-1": []byte("notes cover")},
		}
		syncer := devicesync.NewSyncer(devicesync.NewSyncerInput{BookFileStore: bookFileStore, BookStore: bookStore, DeviceStore: deviceStore, DocumentStore: documentStore, Remarkable: tablet})
		book, err := bookStore.Create(models.Book{Title: "Dune"})
		So(err, ShouldBeNil)
		_, err = syncer.SyncDevice(device.ID)
		So(err, ShouldBeNil)
		Convey("When the first sync runs", func() {
			Convey("Then only the PDF's cover is fetched and stored", func() {
				So(tablet.coverRequests, ShouldResemble, []remarkable.CoverRequest{{DocumentUUID: "document-1", PageID: "page-1"}})
				image, err := documentStore.GetCover(device.ID, "document-1")
				So(err, ShouldBeNil)
				So(string(image), ShouldEqual, "dune cover")
			})
			Convey("Then the book it auto-linked to offers the tablet's cover", func() {
				linked, err := bookStore.Get(book.ID)
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
				image, err := documentStore.GetCover(device.ID, "document-1")
				So(err, ShouldBeNil)
				So(string(image), ShouldEqual, "new cover")
			})
		})
	})
	Convey("Given a PDF whose cover the tablet hasn't rendered yet", t, func() {
		stores := storetest.OpenStores(t)
		bookStore, deviceStore, documentStore, bookFileStore := stores.BookStore, stores.DeviceStore, stores.DocumentStore, stores.BookFileStore
		device, err := deviceStore.Create(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		tablet := &fakeTablet{documents: []models.RemarkableDocument{
			{UUID: "document-1", Title: "Dune", FileType: models.FileTypePDF, LastModified: time.Now().UTC(), CoverPageID: "page-1"},
		}}
		syncer := devicesync.NewSyncer(devicesync.NewSyncerInput{BookFileStore: bookFileStore, BookStore: bookStore, DeviceStore: deviceStore, DocumentStore: documentStore, Remarkable: tablet})
		_, err = syncer.SyncDevice(device.ID)
		So(err, ShouldBeNil)
		Convey("When it's rendered before the next sync", func() {
			tablet.covers = map[string][]byte{"document-1": []byte("dune cover")}
			_, err := syncer.SyncDevice(device.ID)
			Convey("Then that sync picks it up", func() {
				So(err, ShouldBeNil)
				image, err := documentStore.GetCover(device.ID, "document-1")
				So(err, ShouldBeNil)
				So(string(image), ShouldEqual, "dune cover")
			})
		})
	})
}

func TestTabletIdentity(t *testing.T) {
	Convey("Given a device registered before host keys were recorded", t, func() {
		stores := storetest.OpenStores(t)
		bookStore, deviceStore, documentStore, bookFileStore := stores.BookStore, stores.DeviceStore, stores.DocumentStore, stores.BookFileStore
		pairedAt := time.Now().UTC()
		device, err := deviceStore.Create(models.Device{Name: "Paper Pro", Host: "10.0.0.5", PairedAt: &pairedAt})
		So(err, ShouldBeNil)
		tablet := &fakeTablet{hostKey: "ssh-ed25519 AAAA-real-tablet"}
		syncer := devicesync.NewSyncer(devicesync.NewSyncerInput{BookFileStore: bookFileStore, BookStore: bookStore, DeviceStore: deviceStore, DocumentStore: documentStore, Remarkable: tablet})
		Convey("When it syncs", func() {
			_, err := syncer.SyncDevice(device.ID)
			Convey("Then the host key it presented is recorded and required from then on", func() {
				So(err, ShouldBeNil)
				stored, err := deviceStore.Get(device.ID)
				So(err, ShouldBeNil)
				So(stored.HostKey, ShouldEqual, "ssh-ed25519 AAAA-real-tablet")
				_, err = syncer.SyncDevice(device.ID)
				So(err, ShouldBeNil)
				So(tablet.lastTarget.HostKey, ShouldEqual, "ssh-ed25519 AAAA-real-tablet")
			})
		})
		Convey("And later something presents a different host key", func() {
			_, err := syncer.SyncDevice(device.ID)
			So(err, ShouldBeNil)
			tablet.listErr = remarkable.ErrHostKeyChanged
			_, err = syncer.SyncDevice(device.ID)
			Convey("Then the sync fails and the device is unpaired with its identity flagged", func() {
				So(errors.Is(err, remarkable.ErrHostKeyChanged), ShouldBeTrue)
				stored, err := deviceStore.Get(device.ID)
				So(err, ShouldBeNil)
				So(stored.PairedAt, ShouldBeNil)
				So(stored.IdentityChanged, ShouldBeTrue)
			})
			Convey("Then pairing again still requires the recorded host key", func() {
				tablet.pairErr = remarkable.ErrHostKeyChanged
				_, err := syncer.PairDevice(devicesync.PairDeviceInput{DeviceID: device.ID, Password: "secret"})
				So(errors.Is(err, remarkable.ErrHostKeyChanged), ShouldBeTrue)
				So(tablet.lastTarget.HostKey, ShouldEqual, "ssh-ed25519 AAAA-real-tablet")
			})
			Convey("Then pairing with the new identity accepted records the new host key", func() {
				tablet.hostKey = "ssh-ed25519 AAAA-reset-tablet"
				repaired, err := syncer.PairDevice(devicesync.PairDeviceInput{DeviceID: device.ID, Password: "secret", AcceptNewIdentity: true})
				So(err, ShouldBeNil)
				So(tablet.lastTarget.HostKey, ShouldBeEmpty)
				So(repaired.PairedAt, ShouldNotBeNil)
				So(repaired.IdentityChanged, ShouldBeFalse)
				So(repaired.HostKey, ShouldEqual, "ssh-ed25519 AAAA-reset-tablet")
			})
		})
	})
}

func TestPairing(t *testing.T) {
	Convey("Given a syncer with no devices", t, func() {
		stores := storetest.OpenStores(t)
		bookStore, deviceStore, documentStore, bookFileStore := stores.BookStore, stores.DeviceStore, stores.DocumentStore, stores.BookFileStore
		tablet := &fakeTablet{}
		syncer := devicesync.NewSyncer(devicesync.NewSyncerInput{BookFileStore: bookFileStore, BookStore: bookStore, DeviceStore: deviceStore, DocumentStore: documentStore, Remarkable: tablet})
		register := func() (models.Device, error) {
			return syncer.RegisterDevice(devicesync.RegisterDeviceInput{Name: " Paper Pro ", Host: "10.0.0.5", Password: "secret"})
		}
		Convey("When registering a tablet with its password", func() {
			device, err := register()
			Convey("Then it pairs with the password and saves the device as paired", func() {
				So(err, ShouldBeNil)
				So(tablet.pairedWith, ShouldEqual, "secret")
				So(device.Name, ShouldEqual, "Paper Pro")
				stored, err := deviceStore.Get(device.ID)
				So(err, ShouldBeNil)
				So(stored.PairedAt, ShouldNotBeNil)
			})
			Convey("And the tablet later rejects the server's key", func() {
				tablet.listErr = remarkable.ErrNotPaired
				_, err := syncer.SyncDevice(device.ID)
				Convey("Then the sync fails and the device is marked unpaired", func() {
					So(errors.Is(err, remarkable.ErrNotPaired), ShouldBeTrue)
					stored, err := deviceStore.Get(device.ID)
					So(err, ShouldBeNil)
					So(stored.PairedAt, ShouldBeNil)
				})
				Convey("Then pairing it again marks it paired", func() {
					tablet.listErr = nil
					repaired, err := syncer.PairDevice(devicesync.PairDeviceInput{DeviceID: device.ID, Password: "secret"})
					So(err, ShouldBeNil)
					So(repaired.PairedAt, ShouldNotBeNil)
				})
			})
			Convey("And the tablet is just asleep", func() {
				tablet.listErr = remarkable.ErrTabletUnreachable
				_, err := syncer.SyncDevice(device.ID)
				Convey("Then the device stays paired", func() {
					So(errors.Is(err, remarkable.ErrTabletUnreachable), ShouldBeTrue)
					stored, err := deviceStore.Get(device.ID)
					So(err, ShouldBeNil)
					So(stored.PairedAt, ShouldNotBeNil)
				})
			})
		})
		Convey("When registering with a password the tablet rejects", func() {
			tablet.pairErr = remarkable.ErrWrongPassword
			_, err := register()
			Convey("Then it returns ErrWrongPassword and saves no device", func() {
				So(errors.Is(err, remarkable.ErrWrongPassword), ShouldBeTrue)
				devices, err := deviceStore.List()
				So(err, ShouldBeNil)
				So(devices, ShouldBeEmpty)
			})
		})
		Convey("When registering an address that's already registered", func() {
			_, err := register()
			So(err, ShouldBeNil)
			tablet.pairedWith = ""
			_, err = syncer.RegisterDevice(devicesync.RegisterDeviceInput{Name: "Paper Pro again", Host: "10.0.0.5", Password: "secret"})
			Convey("Then it returns ErrDeviceAlreadyRegistered without using the password", func() {
				So(errors.Is(err, devicesync.ErrDeviceAlreadyRegistered), ShouldBeTrue)
				var alreadyRegistered devicesync.DeviceAlreadyRegisteredError
				So(errors.As(err, &alreadyRegistered), ShouldBeTrue)
				So(alreadyRegistered.Name, ShouldEqual, "Paper Pro")
				So(tablet.pairedWith, ShouldBeEmpty)
				devices, err := deviceStore.List()
				So(err, ShouldBeNil)
				So(devices, ShouldHaveLength, 1)
			})
		})
		Convey("When registering without a password", func() {
			_, err := syncer.RegisterDevice(devicesync.RegisterDeviceInput{Name: "Paper Pro", Host: "10.0.0.5"})
			Convey("Then it returns ErrPasswordRequired without contacting the tablet", func() {
				So(errors.Is(err, devicesync.ErrPasswordRequired), ShouldBeTrue)
				So(tablet.pairedWith, ShouldBeEmpty)
			})
		})
		Convey("When registering without a host", func() {
			_, err := syncer.RegisterDevice(devicesync.RegisterDeviceInput{Name: "Paper Pro", Password: "secret"})
			Convey("Then it returns ErrDeviceDetailsRequired", func() {
				So(errors.Is(err, devicesync.ErrDeviceDetailsRequired), ShouldBeTrue)
			})
		})
	})
}

func TestSyncCopiesBooks(t *testing.T) {
	Convey("Given a paired tablet and a book with a file saved on the server", t, func() {
		stores := storetest.OpenStores(t)
		device, err := stores.DeviceStore.Create(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		book, err := stores.BookStore.Create(models.Book{Title: "Pride and Prejudice"})
		So(err, ShouldBeNil)
		_, err = stores.BookFileStore.Save(context.Background(), bookfilestore.SaveInput{
			BookID: book.ID, Source: models.BookFileSourceUpload, Body: strings.NewReader("%PDF-1.7 the book"),
		})
		So(err, ShouldBeNil)
		tablet := &fakeTablet{}
		syncer := devicesync.NewSyncer(devicesync.NewSyncerInput{
			BookFileStore: stores.BookFileStore,
			BookStore:     stores.BookStore,
			DeviceStore:   stores.DeviceStore,
			DocumentStore: stores.DocumentStore,
			Remarkable:    tablet,
		})
		Convey("When syncing in the background", func() {
			_, err := syncer.SyncDevice(device.ID)
			So(err, ShouldBeNil)
			Convey("Then the file is copied to the tablet under the book's title", func() {
				So(tablet.copied, ShouldHaveLength, 1)
				So(tablet.copied[0].input.Title, ShouldEqual, "Pride and Prejudice")
				So(tablet.copied[0].input.FileType, ShouldEqual, models.FileTypePDF)
				So(tablet.copied[0].contents, ShouldEqual, "%PDF-1.7 the book")
			})
			Convey("Then the copy is linked to the book", func() {
				documents, err := stores.DocumentStore.ListByBook(book.ID)
				So(err, ShouldBeNil)
				So(documents, ShouldHaveLength, 1)
				So(documents[0].UUID, ShouldEqual, tablet.copied[0].input.UUID)
			})
			Convey("Then the reading app isn't restarted, leaving the copy waiting to load", func() {
				So(tablet.restarts, ShouldEqual, 0)
				file, err := stores.BookFileStore.Get(book.ID)
				So(err, ShouldBeNil)
				So(file.Deliveries, ShouldHaveLength, 1)
				So(file.Deliveries[0].LoadedAt, ShouldBeNil)
			})
			Convey("And syncing again", func() {
				_, err := syncer.SyncDevice(device.ID)
				So(err, ShouldBeNil)
				Convey("Then it isn't copied twice", func() {
					So(tablet.copied, ShouldHaveLength, 1)
				})
			})
			Convey("And the user syncing now", func() {
				_, err := syncer.SyncDeviceNow(device.ID)
				So(err, ShouldBeNil)
				Convey("Then the reading app restarts to load the copy", func() {
					So(tablet.restarts, ShouldEqual, 1)
					file, err := stores.BookFileStore.Get(book.ID)
					So(err, ShouldBeNil)
					So(file.Deliveries[0].LoadedAt, ShouldNotBeNil)
				})
				Convey("And syncing now once more", func() {
					_, err := syncer.SyncDeviceNow(device.ID)
					So(err, ShouldBeNil)
					Convey("Then the reading app isn't restarted again", func() {
						So(tablet.restarts, ShouldEqual, 1)
					})
				})
			})
		})
		Convey("When the book is already on the tablet", func() {
			tablet.documents = []models.RemarkableDocument{{UUID: "document-1", Title: "Pride and Prejudice", FileType: models.FileTypeEPUB}}
			_, err := syncer.SyncDevice(device.ID)
			So(err, ShouldBeNil)
			_, err = syncer.SyncDeviceNow(device.ID)
			So(err, ShouldBeNil)
			Convey("Then sync records the document linked by title instead of copying the file", func() {
				So(tablet.copied, ShouldBeEmpty)
				So(tablet.restarts, ShouldEqual, 0)
				file, err := stores.BookFileStore.Get(book.ID)
				So(err, ShouldBeNil)
				So(file.Deliveries, ShouldHaveLength, 1)
				So(file.Deliveries[0].DocumentUUID, ShouldEqual, "document-1")
			})
		})
		Convey("When copying fails", func() {
			tablet.copyErr = errors.New("connection reset")
			_, err := syncer.SyncDeviceNow(device.ID)
			Convey("Then the sync still succeeds, and the file waits for the next sync", func() {
				So(err, ShouldBeNil)
				So(tablet.restarts, ShouldEqual, 0)
				undelivered, err := stores.BookFileStore.ListUndelivered(device.ID)
				So(err, ShouldBeNil)
				So(undelivered, ShouldHaveLength, 1)
			})
		})
	})
}
