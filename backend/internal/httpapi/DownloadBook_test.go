package httpapi_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/httpapi"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/internetarchive"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/storetest"
)

type fakeInternetArchive struct {
	ebook       internetarchive.Ebook
	contents    string
	err         error
	identifiers *[]string
}

func (fake fakeInternetArchive) FindEbook(identifiers []string) (internetarchive.Ebook, error) {
	if fake.identifiers != nil {
		*fake.identifiers = identifiers
	}
	return fake.ebook, fake.err
}

func (fake fakeInternetArchive) OpenEbook(ctx context.Context, ebook internetarchive.Ebook) (internetarchive.Download, error) {
	return internetarchive.Download{Body: io.NopCloser(strings.NewReader(fake.contents)), ContentLength: int64(len(fake.contents))}, nil
}

func downloadBook(t *testing.T, openLibrary openlibrary.API, internetArchive internetarchive.API) *httptest.ResponseRecorder {
	stores := storetest.OpenStores(t)
	handler := httpapi.NewHandler(httpapi.NewHandlerInput{
		BookStore:       stores.BookStore,
		DeviceStore:     stores.DeviceStore,
		DocumentStore:   stores.DocumentStore,
		InternetArchive: internetArchive,
		OpenLibrary:     openLibrary,
		SourceStore:     stores.SourceStore,
	})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/search/books/OL66554W/download?title=Pride+and+Prejudice", nil))
	return recorder
}

func TestDownloadBook(t *testing.T) {
	Convey("Given a public domain work with an EPUB on the Internet Archive", t, func() {
		var identifiers []string
		openLibrary := fakeOpenLibrary{scans: openlibrary.Scans{Title: "Pride and Prejudice: A Novel", ArchiveIDs: []string{"prideprejudice00aust"}}}
		internetArchive := fakeInternetArchive{
			ebook:       internetarchive.Ebook{Identifier: "prideprejudice00aust", FileName: "prideprejudice00aust.epub", Format: internetarchive.FormatEPUB},
			contents:    "epub contents",
			identifiers: &identifiers,
		}
		Convey("When downloading it", func() {
			recorder := downloadBook(t, openLibrary, internetArchive)
			Convey("Then it looks for an ebook among the work's scans", func() {
				So(identifiers, ShouldResemble, []string{"prideprejudice00aust"})
			})
			Convey("Then it streams the EPUB as an attachment named after the title", func() {
				So(recorder.Code, ShouldEqual, http.StatusOK)
				So(recorder.Body.String(), ShouldEqual, "epub contents")
				So(recorder.Header().Get("Content-Type"), ShouldEqual, "application/epub+zip")
				So(recorder.Header().Get("Content-Length"), ShouldEqual, "13")
				So(recorder.Header().Get("Content-Disposition"), ShouldEqual, `attachment; filename="Pride and Prejudice A Novel.epub"`)
			})
		})
	})
	Convey("Given a work that isn't in the public domain", t, func() {
		openLibrary := fakeOpenLibrary{err: openlibrary.ErrNotPublicDomain}
		Convey("When downloading it", func() {
			recorder := downloadBook(t, openLibrary, fakeInternetArchive{})
			Convey("Then it responds 404, as no source has it", func() {
				So(recorder.Code, ShouldEqual, http.StatusNotFound)
				So(recorder.Body.String(), ShouldContainSubstring, "none of your ebook sources has this book")
			})
		})
	})
	Convey("Given a public domain work whose scans have no ebook", t, func() {
		openLibrary := fakeOpenLibrary{scans: openlibrary.Scans{Title: "Dune", ArchiveIDs: []string{"dune0000herb"}}}
		internetArchive := fakeInternetArchive{err: internetarchive.ErrNoPublicEbook}
		Convey("When downloading it", func() {
			recorder := downloadBook(t, openLibrary, internetArchive)
			Convey("Then it responds 404, as no source has it", func() {
				So(recorder.Code, ShouldEqual, http.StatusNotFound)
				So(recorder.Body.String(), ShouldContainSubstring, "none of your ebook sources has this book")
			})
		})
	})
}
