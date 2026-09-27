package openlibrary_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ansel1/merry"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
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
					"cover_i": 11481354,
					"number_of_pages_median": 608
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
					PageCount:     608,
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

func TestEditionPageCount(t *testing.T) {
	Convey("Given an Open Library client", t, func() {
		client := openlibrary.NewClient()
		Convey("When looking up an empty ISBN", func() {
			pageCount, err := client.EditionPageCount("")
			Convey("Then it returns ErrEmptyISBN without making a request", func() {
				So(pageCount, ShouldEqual, 0)
				So(merry.Is(err, openlibrary.ErrEmptyISBN), ShouldBeTrue)
			})
		})
	})
	Convey("Given a fake Open Library server that redirects an ISBN to its edition", t, func() {
		server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			switch request.URL.Path {
			case "/isbn/9780441013593.json":
				http.Redirect(responseWriter, request, "/books/OL7353617M.json", http.StatusFound)
			case "/books/OL7353617M.json":
				_, _ = responseWriter.Write([]byte(`{"title": "Dune", "number_of_pages": 544}`))
			case "/isbn/9780000000002.json":
				_, _ = responseWriter.Write([]byte(`{"title": "No page count"}`))
			default:
				http.NotFound(responseWriter, request)
			}
		}))
		Reset(server.Close)
		client := openlibrary.NewClientWithBaseURL(server.URL)
		Convey("When looking up a known ISBN", func() {
			pageCount, err := client.EditionPageCount("9780441013593")
			Convey("Then it follows the redirect and returns the edition's page count", func() {
				So(err, ShouldBeNil)
				So(pageCount, ShouldEqual, 544)
			})
		})
		Convey("When the edition has no page count", func() {
			pageCount, err := client.EditionPageCount("9780000000002")
			Convey("Then it returns 0 without an error", func() {
				So(err, ShouldBeNil)
				So(pageCount, ShouldEqual, 0)
			})
		})
		Convey("When looking up an unknown ISBN", func() {
			pageCount, err := client.EditionPageCount("9780000000001")
			Convey("Then it returns ErrEditionNotFound", func() {
				So(pageCount, ShouldEqual, 0)
				So(merry.Is(err, openlibrary.ErrEditionNotFound), ShouldBeTrue)
			})
		})
	})
}
