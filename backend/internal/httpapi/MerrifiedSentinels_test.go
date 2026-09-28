package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/httpapi"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/store"
)

func TestStoreErrorResponses(t *testing.T) {
	Convey("Given the API holding a 544-page book", t, func() {
		opened, err := store.Open(filepath.Join(t.TempDir(), "books.db"))
		So(err, ShouldBeNil)
		Reset(func() { _ = opened.Close() })
		pageCount := 544
		book, err := opened.Books.Create(models.Book{Title: "Dune", PageCount: &pageCount})
		So(err, ShouldBeNil)
		handler := httpapi.NewHandler(httpapi.NewHandlerInput{Store: opened, OpenLibrary: fakeOpenLibrary{}})
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
	})
}
