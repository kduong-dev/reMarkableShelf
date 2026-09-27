package openlibrary_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ansel1/merry"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/kqvd/reMarkableShelf/backend/internal/openlibrary"
)

func TestSearch(t *testing.T) {
	Convey("Given an Open Library client", t, func() {
		client := openlibrary.NewClient()
		Convey("When searching with an empty query", func() {
			results, err := client.Search("")
			Convey("Then it returns ErrEmptyQuery without making a request", func() {
				So(results, ShouldBeNil)
				So(merry.Is(err, openlibrary.ErrEmptyQuery), ShouldBeTrue)
			})
		})
	})
	Convey("Given a fake Open Library server returning a non-200 status", t, func() {
		server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			responseWriter.WriteHeader(http.StatusTooManyRequests)
		}))
		Reset(server.Close)
		client := openlibrary.NewClientWithBaseURL(server.URL)
		Convey("When searching", func() {
			results, err := client.Search("dune")
			Convey("Then it returns ErrUpstreamUnavailable", func() {
				So(results, ShouldBeNil)
				So(merry.Is(err, openlibrary.ErrUpstreamUnavailable), ShouldBeTrue)
			})
		})
	})
	Convey("Given a fake Open Library server returning a matching work", t, func() {
		var receivedUserAgent string
		server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			receivedUserAgent = request.Header.Get("User-Agent")
			responseWriter.Header().Set("Content-Type", "application/json")
			_, _ = responseWriter.Write([]byte(`{
				"docs": [{
					"key": "/works/OL893415W",
					"title": "Dune",
					"author_name": ["Frank Herbert"],
					"isbn": ["0441013597", "9780441013593"],
					"cover_i": 11481354
				}, {
					"key": "/works/OL1W",
					"title": "Untitled"
				}]
			}`))
		}))
		Reset(server.Close)
		client := openlibrary.NewClientWithBaseURL(server.URL)
		Convey("When searching", func() {
			results, err := client.Search("dune")
			Convey("Then it identifies itself with a User-Agent", func() {
				So(receivedUserAgent, ShouldContainSubstring, "reMarkableShelf")
			})
			Convey("Then it maps the work into a Result, preferring a 13-digit ISBN", func() {
				So(err, ShouldBeNil)
				So(results, ShouldHaveLength, 2)
				So(results[0], ShouldResemble, openlibrary.Result{
					OpenLibraryID: "OL893415W",
					Title:         "Dune",
					Author:        "Frank Herbert",
					ISBN:          "9780441013593",
					CoverURL:      "https://covers.openlibrary.org/b/id/11481354-M.jpg",
				})
			})
			Convey("Then a work without author, ISBN or cover leaves those fields empty", func() {
				So(results[1], ShouldResemble, openlibrary.Result{
					OpenLibraryID: "OL1W",
					Title:         "Untitled",
				})
			})
		})
	})
}
