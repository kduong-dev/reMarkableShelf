package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/httpapi"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/store"
)

func TestSearchBooks(t *testing.T) {
	Convey("Given the API with a fake Open Library", t, func() {
		opened, err := store.Open(filepath.Join(t.TempDir(), "books.db"))
		So(err, ShouldBeNil)
		Reset(func() { _ = opened.Close() })
		var searchInput openlibrary.SearchInput
		handler := httpapi.NewHandler(httpapi.NewHandlerInput{
			Store:       opened,
			OpenLibrary: fakeOpenLibrary{searchInput: &searchInput},
		})
		search := func(query string) *httptest.ResponseRecorder {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/search/books?"+query, nil))
			return recorder
		}
		Convey("When searching with filters", func() {
			recorder := search("q=le+guin&sort=rating&language=eng&publishedFrom=1960&publishedTo=1979")
			Convey("Then they are passed to Open Library", func() {
				So(recorder.Code, ShouldEqual, http.StatusOK)
				So(searchInput, ShouldResemble, openlibrary.SearchInput{
					Query:         "le guin",
					Sort:          openlibrary.SortRating,
					Language:      "eng",
					PublishedFrom: 1960,
					PublishedTo:   1979,
				})
			})
		})
		Convey("When a published year isn't a number", func() {
			recorder := search("q=le+guin&publishedFrom=sixties")
			Convey("Then it responds 400 naming the parameter", func() {
				So(recorder.Code, ShouldEqual, http.StatusBadRequest)
				So(recorder.Body.String(), ShouldContainSubstring, "publishedFrom must be a year")
			})
		})
	})
}
