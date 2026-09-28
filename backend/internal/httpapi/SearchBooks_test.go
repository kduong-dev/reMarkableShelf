package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/httpapi"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/storetest"
)

func TestSearchBooks(t *testing.T) {
	Convey("Given the API with a fake Open Library", t, func() {
		bookStore, deviceStore, documentStore := storetest.Open(t)
		var searchInput openlibrary.SearchInput
		handler := httpapi.NewHandler(httpapi.NewHandlerInput{
			BookStore:     bookStore,
			DeviceStore:   deviceStore,
			DocumentStore: documentStore,
			OpenLibrary:   fakeOpenLibrary{searchInput: &searchInput},
		})
		search := func(query string) *httptest.ResponseRecorder {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/search/books?"+query, nil))
			return recorder
		}
		Convey("When searching with filters", func() {
			recorder := search("q=le+guin&subject=fantasy&sort=rating&language=eng&publishedFrom=1960&publishedTo=1979&page=2")
			Convey("Then they are passed to Open Library", func() {
				So(recorder.Code, ShouldEqual, http.StatusOK)
				So(searchInput, ShouldResemble, openlibrary.SearchInput{
					Query:         "le guin",
					Subject:       "fantasy",
					Sort:          openlibrary.SortRating,
					Language:      "eng",
					PublishedFrom: 1960,
					PublishedTo:   1979,
					Page:          2,
				})
			})
		})
		Convey("When a published year isn't a number", func() {
			recorder := search("q=le+guin&publishedFrom=sixties")
			Convey("Then it responds 400 naming the parameter", func() {
				So(recorder.Code, ShouldEqual, http.StatusBadRequest)
				So(recorder.Body.String(), ShouldContainSubstring, "publishedFrom must be a whole number")
			})
		})
	})
}
