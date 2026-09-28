package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ansel1/merry"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicesync"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/httpapi"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/storetest"
)

func TestErrorResponses(t *testing.T) {
	Convey("Given the API holding a 544-page book", t, func() {
		bookStore, deviceStore, documentStore := storetest.Open(t)
		pageCount := 544
		book, err := bookStore.Create(models.Book{Title: "Dune", PageCount: &pageCount})
		So(err, ShouldBeNil)
		device, err := deviceStore.Create(models.Device{Name: "Paper Pro", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		// Every request below fails before reaching a tablet.
		handler := httpapi.NewHandler(httpapi.NewHandlerInput{
			BookStore:     bookStore,
			DeviceStore:   deviceStore,
			DocumentStore: documentStore,
			OpenLibrary:   fakeOpenLibrary{},
			Syncer:        devicesync.NewSyncer(devicesync.NewSyncerInput{BookStore: bookStore, DeviceStore: deviceStore, DocumentStore: documentStore}),
		})
		send := func(method, path, body string) (int, string) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))
			var message httpx.ResponseMessage
			So(json.Unmarshal(recorder.Body.Bytes(), &message), ShouldBeNil)
			return recorder.Code, message.Message
		}
		Convey("When getting a book that doesn't exist", func() {
			statusCode, message := send(http.MethodGet, "/api/books/missing", "")
			Convey("Then it responds not found", func() {
				So(statusCode, ShouldEqual, http.StatusNotFound)
				So(message, ShouldEqual, "no book with that id")
			})
		})
		Convey("When bookmarking a page past the end of the book", func() {
			statusCode, message := send(http.MethodPut, "/api/books/"+book.ID, `{"currentPage": 545}`)
			Convey("Then it responds bad request", func() {
				So(statusCode, ShouldEqual, http.StatusBadRequest)
				So(message, ShouldEqual, "the page can't be negative or past the end of the book")
			})
		})
		Convey("When renaming a device that doesn't exist", func() {
			statusCode, message := send(http.MethodPut, "/api/devices/missing", `{"name": "Paper Pro"}`)
			Convey("Then it responds not found", func() {
				So(statusCode, ShouldEqual, http.StatusNotFound)
				So(message, ShouldEqual, "no device with that id")
			})
		})
		Convey("When getting the cover of a document that has none", func() {
			statusCode, message := send(http.MethodGet, "/api/devices/missing/documents/missing/cover", "")
			Convey("Then it responds not found", func() {
				So(statusCode, ShouldEqual, http.StatusNotFound)
				So(message, ShouldEqual, "that document has no synced cover")
			})
		})
		Convey("When registering a device without a password", func() {
			statusCode, message := send(http.MethodPost, "/api/devices", `{"name": "Move", "host": "10.0.0.6"}`)
			Convey("Then it responds bad request", func() {
				So(statusCode, ShouldEqual, http.StatusBadRequest)
				So(message, ShouldEqual, "enter the tablet's password to pair it")
			})
		})
		Convey("When registering the host of a device already registered", func() {
			statusCode, message := send(http.MethodPost, "/api/devices", `{"name": "Move", "host": "`+device.Host+`", "password": "secret"}`)
			Convey("Then it responds conflict, naming the registered device", func() {
				So(statusCode, ShouldEqual, http.StatusConflict)
				So(message, ShouldContainSubstring, `already registered as "Paper Pro"`)
			})
		})
	})
	Convey("Given Open Library fails a search", t, func() {
		bookStore, deviceStore, documentStore := storetest.Open(t)
		search := func(openLibraryErr error) (int, string) {
			handler := httpapi.NewHandler(httpapi.NewHandlerInput{BookStore: bookStore, DeviceStore: deviceStore, DocumentStore: documentStore, OpenLibrary: fakeOpenLibrary{err: openLibraryErr}})
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/search/books?q=dune", nil))
			var message httpx.ResponseMessage
			So(json.Unmarshal(recorder.Body.Bytes(), &message), ShouldBeNil)
			return recorder.Code, message.Message
		}
		Convey("When the query is too short", func() {
			statusCode, message := search(openlibrary.ErrQueryTooShort)
			Convey("Then it responds bad request", func() {
				So(statusCode, ShouldEqual, http.StatusBadRequest)
				So(message, ShouldEqual, "search for at least 3 characters")
			})
		})
		Convey("When Open Library responds with an error of its own", func() {
			upstream := merry.New("request failed with status 401").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("Unauthorized")
			statusCode, message := search(fmt.Errorf("%w: %w", openlibrary.ErrUpstreamUnavailable, upstream))
			Convey("Then it responds bad gateway rather than passing Open Library's status on", func() {
				So(statusCode, ShouldEqual, http.StatusBadGateway)
				So(message, ShouldEqual, "Open Library is unavailable, try again shortly")
			})
		})
	})
}
