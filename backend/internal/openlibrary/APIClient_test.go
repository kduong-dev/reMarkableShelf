package openlibrary_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ansel1/merry"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
)

func TestSearch(t *testing.T) {
	Convey("Given an Open Library client", t, func() {
		client := openlibrary.NewClient()
		Convey("When searching with an empty query", func() {
			results, err := client.Search(openlibrary.SearchInput{Query: ""})
			Convey("Then it returns ErrEmptyQuery without making a request", func() {
				So(results, ShouldBeNil)
				So(merry.Is(err, openlibrary.ErrEmptyQuery), ShouldBeTrue)
			})
		})
	})
	Convey("Given an Open Library client", t, func() {
		client := openlibrary.NewClient()
		Convey("When searching with fewer than 3 characters around whitespace", func() {
			results, err := client.Search(openlibrary.SearchInput{Query: "  ab  "})
			Convey("Then it returns ErrQueryTooShort without making a request", func() {
				So(results, ShouldBeNil)
				So(merry.Is(err, openlibrary.ErrQueryTooShort), ShouldBeTrue)
				So(merry.HTTPCode(err), ShouldEqual, http.StatusBadRequest)
				So(merry.UserMessage(err), ShouldEqual, "search for at least 3 characters")
			})
		})
	})
	Convey("Given a fake Open Library server that rejects the query", t, func() {
		server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			responseWriter.WriteHeader(http.StatusUnprocessableEntity)
		}))
		Reset(server.Close)
		client := openlibrary.NewClientWithBaseURL(server.URL)
		Convey("When searching", func() {
			results, err := client.Search(openlibrary.SearchInput{Query: "the"})
			Convey("Then it returns ErrQueryRejected as a client error", func() {
				So(results, ShouldBeNil)
				So(merry.Is(err, openlibrary.ErrQueryRejected), ShouldBeTrue)
				So(merry.HTTPCode(err), ShouldEqual, http.StatusBadRequest)
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
			results, err := client.Search(openlibrary.SearchInput{Query: "dune"})
			Convey("Then it returns ErrUpstreamUnavailable, keeping the upstream status for the logs", func() {
				So(results, ShouldBeNil)
				So(merry.Is(err, openlibrary.ErrUpstreamUnavailable), ShouldBeTrue)
				So(merry.HTTPCode(err), ShouldEqual, http.StatusBadGateway)
				So(fmt.Sprintf("%v", err), ShouldContainSubstring, "status 429")
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
					"number_of_pages_median": 608,
					"first_publish_year": 1965,
					"ratings_average": 4.26785
				}, {
					"key": "/works/OL1W",
					"title": "Untitled"
				}]
			}`))
		}))
		Reset(server.Close)
		client := openlibrary.NewClientWithBaseURL(server.URL)
		Convey("When searching", func() {
			results, err := client.Search(openlibrary.SearchInput{Query: "dune"})
			Convey("Then it identifies itself with a User-Agent", func() {
				So(receivedUserAgent, ShouldContainSubstring, "reMarkableShelf")
			})
			Convey("Then it maps the work into a Result, preferring a 13-digit ISBN", func() {
				So(err, ShouldBeNil)
				So(results, ShouldHaveLength, 2)
				So(results[0], ShouldResemble, openlibrary.Result{
					OpenLibraryID:    "OL893415W",
					Title:            "Dune",
					Author:           "Frank Herbert",
					ISBN:             "9780441013593",
					CoverURL:         "https://covers.openlibrary.org/b/id/11481354-M.jpg",
					PageCount:        608,
					FirstPublishYear: 1965,
					AverageRating:    4.3,
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

func TestSearchFilters(t *testing.T) {
	Convey("Given a fake Open Library server recording the search request", t, func() {
		var received url.Values
		server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			received = request.URL.Query()
			_, _ = responseWriter.Write([]byte(`{"docs": []}`))
		}))
		Reset(server.Close)
		client := openlibrary.NewClientWithBaseURL(server.URL)
		Convey("When searching without filters", func() {
			_, err := client.Search(openlibrary.SearchInput{Query: "le guin"})
			Convey("Then it sends the bare query with Open Library's default sort", func() {
				So(err, ShouldBeNil)
				So(received.Get("q"), ShouldEqual, "le guin")
				So(received.Has("sort"), ShouldBeFalse)
			})
		})
		Convey("When searching with every filter", func() {
			_, err := client.Search(openlibrary.SearchInput{
				Query:         "le guin",
				Sort:          openlibrary.SortRating,
				Language:      "eng",
				PublishedFrom: 1960,
				PublishedTo:   1979,
			})
			Convey("Then it appends them as fielded clauses and sets the sort", func() {
				So(err, ShouldBeNil)
				So(received.Get("q"), ShouldEqual, "le guin language:eng first_publish_year:[1960 TO 1979]")
				So(received.Get("sort"), ShouldEqual, "rating")
			})
		})
		Convey("When searching with only a start year", func() {
			_, err := client.Search(openlibrary.SearchInput{Query: "le guin", PublishedFrom: 2010})
			Convey("Then the range is open-ended", func() {
				So(err, ShouldBeNil)
				So(received.Get("q"), ShouldEqual, "le guin first_publish_year:[2010 TO *]")
			})
		})
	})
	Convey("Given an Open Library client", t, func() {
		client := openlibrary.NewClient()
		Convey("When searching with an unknown sort", func() {
			_, err := client.Search(openlibrary.SearchInput{Query: "le guin", Sort: "pages"})
			Convey("Then it returns ErrInvalidSort", func() {
				So(merry.Is(err, openlibrary.ErrInvalidSort), ShouldBeTrue)
			})
		})
		Convey("When searching with a malformed language", func() {
			_, err := client.Search(openlibrary.SearchInput{Query: "le guin", Language: "eng language:fre"})
			Convey("Then it returns ErrInvalidLanguage rather than injecting a clause", func() {
				So(merry.Is(err, openlibrary.ErrInvalidLanguage), ShouldBeTrue)
			})
		})
		Convey("When searching with a reversed year range", func() {
			_, err := client.Search(openlibrary.SearchInput{Query: "le guin", PublishedFrom: 1980, PublishedTo: 1960})
			Convey("Then it returns ErrInvalidYearRange", func() {
				So(merry.Is(err, openlibrary.ErrInvalidYearRange), ShouldBeTrue)
				So(merry.HTTPCode(err), ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}
