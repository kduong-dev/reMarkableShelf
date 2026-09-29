package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/httpapi"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/storetest"
)

type fakeOpenLibrary struct {
	pageCount   int
	scans       openlibrary.Scans
	err         error
	searchInput *openlibrary.SearchInput
}

func (fake fakeOpenLibrary) Search(input openlibrary.SearchInput) (openlibrary.SearchResults, error) {
	if fake.searchInput != nil {
		*fake.searchInput = input
	}
	return openlibrary.SearchResults{}, fake.err
}

func (fake fakeOpenLibrary) EditionPageCount(isbn string) (int, error) {
	return fake.pageCount, fake.err
}

func (fake fakeOpenLibrary) PublicScans(openLibraryID string) (openlibrary.Scans, error) {
	return fake.scans, fake.err
}

func createBook(t *testing.T, openLibrary openlibrary.API, body string) models.Book {
	bookStore, deviceStore, documentStore := storetest.Open(t)
	handler := httpapi.NewHandler(httpapi.NewHandlerInput{BookStore: bookStore, DeviceStore: deviceStore, DocumentStore: documentStore, OpenLibrary: openLibrary})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/books", strings.NewReader(body)))
	So(recorder.Code, ShouldEqual, http.StatusCreated)
	var book models.Book
	So(json.Unmarshal(recorder.Body.Bytes(), &book), ShouldBeNil)
	return book
}

func TestCreateBook(t *testing.T) {
	Convey("Given Open Library knows the edition's page count", t, func() {
		openLibrary := fakeOpenLibrary{pageCount: 544}
		Convey("When creating a book with an ISBN and the median page count", func() {
			book := createBook(t, openLibrary, `{"title": "Dune", "isbn": "9780441013593", "pageCount": 608}`)
			Convey("Then the edition's page count replaces the median", func() {
				So(*book.PageCount, ShouldEqual, 544)
			})
		})
		Convey("When creating a book without an ISBN", func() {
			book := createBook(t, openLibrary, `{"title": "Meeting notes"}`)
			Convey("Then no page count is looked up", func() {
				So(book.PageCount, ShouldBeNil)
			})
		})
	})
	Convey("Given Open Library can't find the edition", t, func() {
		openLibrary := fakeOpenLibrary{err: openlibrary.ErrEditionNotFound}
		Convey("When creating a book with an ISBN and the median page count", func() {
			book := createBook(t, openLibrary, `{"title": "Dune", "isbn": "9780441013593", "pageCount": 608}`)
			Convey("Then the book is still created with the median", func() {
				So(*book.PageCount, ShouldEqual, 608)
			})
		})
	})
	Convey("Given Open Library is unavailable", t, func() {
		openLibrary := fakeOpenLibrary{err: openlibrary.ErrUpstreamUnavailable}
		Convey("When creating a book with an ISBN", func() {
			book := createBook(t, openLibrary, `{"title": "Dune", "isbn": "9780441013593"}`)
			Convey("Then the book is still created without a page count", func() {
				So(book.Title, ShouldEqual, "Dune")
				So(book.PageCount, ShouldBeNil)
			})
		})
	})
}
