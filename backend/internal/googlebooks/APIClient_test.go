package googlebooks_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ansel1/merry"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/kqvd/reMarkableShelf/backend/internal/googlebooks"
)

func TestSearch(t *testing.T) {
	Convey("Given a Google Books client", t, func() {
		client := googlebooks.NewClient()
		Convey("When searching with an empty query", func() {
			results, err := client.Search("")
			Convey("Then it returns ErrEmptyQuery without making a request", func() {
				So(results, ShouldBeNil)
				So(merry.Is(err, googlebooks.ErrEmptyQuery), ShouldBeTrue)
			})
		})
	})
	Convey("Given a fake Google Books server returning a non-200 status", t, func() {
		server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			responseWriter.WriteHeader(http.StatusTooManyRequests)
		}))
		Reset(server.Close)
		client := googlebooks.NewClientWithBaseURL(server.URL)
		Convey("When searching", func() {
			results, err := client.Search("dune")
			Convey("Then it returns ErrUpstreamUnavailable", func() {
				So(results, ShouldBeNil)
				So(merry.Is(err, googlebooks.ErrUpstreamUnavailable), ShouldBeTrue)
			})
		})
	})
	Convey("Given a fake Google Books server returning a matching volume", t, func() {
		server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			responseWriter.Header().Set("Content-Type", "application/json")
			_, _ = responseWriter.Write([]byte(`{
				"items": [{
					"id": "abc123",
					"volumeInfo": {
						"title": "Dune",
						"authors": ["Frank Herbert"],
						"industryIdentifiers": [
							{"type": "ISBN_10", "identifier": "0441013597"},
							{"type": "ISBN_13", "identifier": "9780441013593"}
						],
						"imageLinks": {"thumbnail": "https://example.com/dune.jpg"}
					}
				}]
			}`))
		}))
		Reset(server.Close)
		client := googlebooks.NewClientWithBaseURL(server.URL)
		Convey("When searching", func() {
			results, err := client.Search("dune")
			Convey("Then it maps the volume into a Result, preferring ISBN_13", func() {
				So(err, ShouldBeNil)
				So(results, ShouldHaveLength, 1)
				So(results[0], ShouldResemble, googlebooks.Result{
					GoogleBooksID: "abc123",
					Title:         "Dune",
					Author:        "Frank Herbert",
					ISBN:          "9780441013593",
					CoverURL:      "https://example.com/dune.jpg",
				})
			})
		})
	})
}
