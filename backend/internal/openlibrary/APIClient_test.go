package openlibrary_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
)

func TestSearch(t *testing.T) {
	Convey("Given an Open Library client", t, func() {
		client := openlibrary.NewClient()
		Convey("When searching with an empty query", func() {
			results, err := client.Search(openlibrary.SearchInput{Query: ""})
			Convey("Then it returns ErrEmptyQuery without making a request", func() {
				So(results.Results, ShouldBeEmpty)
				So(errors.Is(err, openlibrary.ErrEmptyQuery), ShouldBeTrue)
			})
		})
	})
	Convey("Given an Open Library client", t, func() {
		client := openlibrary.NewClient()
		Convey("When searching with fewer than 3 characters around whitespace", func() {
			results, err := client.Search(openlibrary.SearchInput{Query: "  ab  "})
			Convey("Then it returns ErrQueryTooShort without making a request", func() {
				So(results.Results, ShouldBeEmpty)
				So(errors.Is(err, openlibrary.ErrQueryTooShort), ShouldBeTrue)
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
				So(results.Results, ShouldBeEmpty)
				So(errors.Is(err, openlibrary.ErrQueryRejected), ShouldBeTrue)
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
				So(results.Results, ShouldBeEmpty)
				So(errors.Is(err, openlibrary.ErrUpstreamUnavailable), ShouldBeTrue)
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
				"numFound": 57,
				"docs": [{
					"key": "/works/OL893415W",
					"title": "Dune",
					"author_name": ["Frank Herbert"],
					"isbn": ["0441013597", "9780441013593"],
					"cover_i": 11481354,
					"number_of_pages_median": 608,
					"first_publish_year": 1965,
					"ratings_average": 4.26785,
					"ebook_access": "borrowable"
				}, {
					"key": "/works/OL1W",
					"title": "Untitled",
					"ebook_access": "public"
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
				So(results.Results, ShouldHaveLength, 2)
				So(results.Total, ShouldEqual, 57)
				So(results.Results[0], ShouldResemble, openlibrary.Result{
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
				So(results.Results[1], ShouldResemble, openlibrary.Result{
					OpenLibraryID: "OL1W",
					Title:         "Untitled",
					Downloadable:  true,
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
				So(errors.Is(err, openlibrary.ErrEmptyISBN), ShouldBeTrue)
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
				So(errors.Is(err, openlibrary.ErrEditionNotFound), ShouldBeTrue)
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
		Convey("When browsing a genre without a query", func() {
			_, err := client.Search(openlibrary.SearchInput{Subject: "science fiction"})
			Convey("Then it searches the quoted subject alone", func() {
				So(err, ShouldBeNil)
				So(received.Get("q"), ShouldEqual, `subject:"science fiction"`)
			})
		})
		Convey("When searching a genre with a query", func() {
			_, err := client.Search(openlibrary.SearchInput{Query: "le guin", Subject: "fantasy"})
			Convey("Then the subject narrows the query", func() {
				So(err, ShouldBeNil)
				So(received.Get("q"), ShouldEqual, `le guin subject:"fantasy"`)
			})
		})
		Convey("When asking for the third page", func() {
			_, err := client.Search(openlibrary.SearchInput{Query: "le guin", Page: 3})
			Convey("Then it requests that page at the page size", func() {
				So(err, ShouldBeNil)
				So(received.Get("page"), ShouldEqual, "3")
				So(received.Get("limit"), ShouldEqual, "20")
			})
		})
		Convey("When asking for the first page", func() {
			_, err := client.Search(openlibrary.SearchInput{Query: "le guin", Page: 1})
			Convey("Then it leaves the page to Open Library's default", func() {
				So(err, ShouldBeNil)
				So(received.Has("page"), ShouldBeFalse)
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
				So(errors.Is(err, openlibrary.ErrInvalidSort), ShouldBeTrue)
			})
		})
		Convey("When searching with a malformed language", func() {
			_, err := client.Search(openlibrary.SearchInput{Query: "le guin", Language: "eng language:fre"})
			Convey("Then it returns ErrInvalidLanguage rather than injecting a clause", func() {
				So(errors.Is(err, openlibrary.ErrInvalidLanguage), ShouldBeTrue)
			})
		})
		Convey("When searching with neither a query nor a subject", func() {
			_, err := client.Search(openlibrary.SearchInput{Sort: openlibrary.SortRating})
			Convey("Then it returns ErrEmptyQuery", func() {
				So(errors.Is(err, openlibrary.ErrEmptyQuery), ShouldBeTrue)
			})
		})
		Convey("When the subject tries to close its quotes and add a clause", func() {
			_, err := client.Search(openlibrary.SearchInput{Subject: `fantasy" OR author:"x`})
			Convey("Then it returns ErrInvalidSubject", func() {
				So(errors.Is(err, openlibrary.ErrInvalidSubject), ShouldBeTrue)
			})
		})
		Convey("When asking for a negative page", func() {
			_, err := client.Search(openlibrary.SearchInput{Query: "le guin", Page: -1})
			Convey("Then it returns ErrInvalidPage", func() {
				So(errors.Is(err, openlibrary.ErrInvalidPage), ShouldBeTrue)
			})
		})
		Convey("When searching with a reversed year range", func() {
			_, err := client.Search(openlibrary.SearchInput{Query: "le guin", PublishedFrom: 1980, PublishedTo: 1960})
			Convey("Then it returns ErrInvalidYearRange", func() {
				So(errors.Is(err, openlibrary.ErrInvalidYearRange), ShouldBeTrue)
			})
		})
	})
}

func TestPublicScans(t *testing.T) {
	Convey("Given an Open Library client", t, func() {
		client := openlibrary.NewClient()
		Convey("When looking up an id that isn't a work's", func() {
			_, err := client.PublicScans(`OL1W" OR title:"x`)
			Convey("Then it returns ErrInvalidWorkID without making a request", func() {
				So(errors.Is(err, openlibrary.ErrInvalidWorkID), ShouldBeTrue)
			})
		})
	})
	Convey("Given a fake Open Library server with public and borrowable works", t, func() {
		var received url.Values
		server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			received = request.URL.Query()
			switch received.Get("q") {
			case `key:"/works/OL66554W"`:
				_, _ = responseWriter.Write([]byte(`{"docs": [{"key": "/works/OL66554W", "title": "Pride and Prejudice", "ebook_access": "public", "ia": ["prideprejudice00aust", "prideprejudice0000jane"]}]}`))
			case `key:"/works/OL893415W"`:
				_, _ = responseWriter.Write([]byte(`{"docs": [{"key": "/works/OL893415W", "title": "Dune", "ebook_access": "borrowable", "ia": ["dune0000herb"]}]}`))
			default:
				_, _ = responseWriter.Write([]byte(`{"docs": []}`))
			}
		}))
		Reset(server.Close)
		client := openlibrary.NewClientWithBaseURL(server.URL)
		Convey("When looking up a public domain work", func() {
			scans, err := client.PublicScans("OL66554W")
			Convey("Then it asks for the work's scans", func() {
				So(received.Get("fields"), ShouldContainSubstring, "ia")
			})
			Convey("Then it returns the title and every scan", func() {
				So(err, ShouldBeNil)
				So(scans, ShouldResemble, openlibrary.Scans{
					Title:      "Pride and Prejudice",
					ArchiveIDs: []string{"prideprejudice00aust", "prideprejudice0000jane"},
				})
			})
		})
		Convey("When looking up a work that can only be borrowed", func() {
			_, err := client.PublicScans("OL893415W")
			Convey("Then it returns ErrNotPublicDomain", func() {
				So(errors.Is(err, openlibrary.ErrNotPublicDomain), ShouldBeTrue)
			})
		})
		Convey("When looking up an unknown work", func() {
			_, err := client.PublicScans("OL1W")
			Convey("Then it returns ErrWorkNotFound", func() {
				So(errors.Is(err, openlibrary.ErrWorkNotFound), ShouldBeTrue)
			})
		})
	})
}
