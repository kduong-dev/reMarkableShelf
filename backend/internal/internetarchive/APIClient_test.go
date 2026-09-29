package internetarchive_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/internetarchive"
)

// fakeArchive serves an advanced search answering with searchBody, and each
// item's file listing from filesByIdentifier.
func fakeArchive(searchBody string, filesByIdentifier map[string]string, searchQuery *string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/advancedsearch.php" {
			if searchQuery != nil {
				*searchQuery = request.URL.Query().Get("q")
			}
			_, _ = responseWriter.Write([]byte(searchBody))
			return
		}
		for identifier, files := range filesByIdentifier {
			if request.URL.Path == "/metadata/"+identifier+"/files" {
				_, _ = responseWriter.Write([]byte(files))
				return
			}
		}
		http.NotFound(responseWriter, request)
	}))
}

func TestFindEbook(t *testing.T) {
	Convey("Given an Internet Archive client", t, func() {
		client := internetarchive.NewClient()
		Convey("When finding an ebook among identifiers that aren't the archive's", func() {
			_, err := client.FindEbook([]string{"a) OR (b"})
			Convey("Then it returns ErrNoPublicEbook without making a request", func() {
				So(errors.Is(err, internetarchive.ErrNoPublicEbook), ShouldBeTrue)
			})
		})
	})
	Convey("Given a fake archive with a scan and a Project Gutenberg transcription", t, func() {
		var searchQuery string
		server := fakeArchive(
			`{"response": {"docs": [{"identifier": "prideprejudice00aust", "collection": ["americana"]}, {"identifier": "pride42671gut", "collection": "gutenberg"}]}}`,
			map[string]string{
				"prideprejudice00aust": `{"result": [{"name": "prideprejudice00aust.pdf", "format": "Text PDF"}, {"name": "prideprejudice00aust.epub", "format": "EPUB"}]}`,
				"pride42671gut":        `{"result": [{"name": "42671.txt", "format": "Text"}, {"name": "pg42671.epub", "format": "EPUB"}]}`,
			},
			&searchQuery,
		)
		Reset(server.Close)
		client := internetarchive.NewClientWithBaseURL(server.URL)
		Convey("When finding an ebook", func() {
			ebook, err := client.FindEbook([]string{"prideprejudice00aust", "prideprejudice0000jane", "pride42671gut"})
			Convey("Then it searches the scans, leaving out borrowable ones", func() {
				So(searchQuery, ShouldEqual, "identifier:(prideprejudice00aust OR prideprejudice0000jane OR pride42671gut) AND mediatype:texts AND NOT access-restricted-item:true")
			})
			Convey("Then it prefers Project Gutenberg's EPUB", func() {
				So(err, ShouldBeNil)
				So(ebook, ShouldResemble, internetarchive.Ebook{Identifier: "pride42671gut", FileName: "pg42671.epub", Format: internetarchive.FormatEPUB})
			})
		})
	})
	Convey("Given a fake archive whose public scans have a PDF but no EPUB", t, func() {
		server := fakeArchive(
			`{"response": {"docs": [{"identifier": "first"}, {"identifier": "second"}]}}`,
			map[string]string{
				"first":  `{"result": [{"name": "first.pdf", "format": "Text PDF"}]}`,
				"second": `{"result": [{"name": "second.epub", "format": "EPUB", "private": "true"}]}`,
			},
			nil,
		)
		Reset(server.Close)
		client := internetarchive.NewClientWithBaseURL(server.URL)
		Convey("When finding an ebook", func() {
			ebook, err := client.FindEbook([]string{"first", "second"})
			Convey("Then it falls back to the PDF, skipping private files", func() {
				So(err, ShouldBeNil)
				So(ebook, ShouldResemble, internetarchive.Ebook{Identifier: "first", FileName: "first.pdf", Format: internetarchive.FormatPDF})
			})
		})
	})
	Convey("Given a fake archive where every scan is borrowable", t, func() {
		server := fakeArchive(`{"response": {"docs": []}}`, nil, nil)
		Reset(server.Close)
		client := internetarchive.NewClientWithBaseURL(server.URL)
		Convey("When finding an ebook", func() {
			_, err := client.FindEbook([]string{"dune0000herb"})
			Convey("Then it returns ErrNoPublicEbook", func() {
				So(errors.Is(err, internetarchive.ErrNoPublicEbook), ShouldBeTrue)
			})
		})
	})
	Convey("Given a fake archive recording each search", t, func() {
		var searches []string
		server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			searches = append(searches, request.URL.Query().Get("q"))
			_, _ = responseWriter.Write([]byte(`{"response": {"docs": []}}`))
		}))
		Reset(server.Close)
		client := internetarchive.NewClientWithBaseURL(server.URL)
		Convey("When finding an ebook among hundreds of scans", func() {
			identifiers := make([]string, 300)
			for index := range identifiers {
				identifiers[index] = fmt.Sprintf("scan%d", index)
			}
			_, err := client.FindEbook(identifiers)
			Convey("Then it searches the first 120 in batches short enough for the archive", func() {
				So(errors.Is(err, internetarchive.ErrNoPublicEbook), ShouldBeTrue)
				So(searches, ShouldHaveLength, 3)
				So(searches[0], ShouldStartWith, "identifier:(scan0 OR scan1 OR ")
				So(searches[2], ShouldContainSubstring, " OR scan119)")
			})
		})
	})
	Convey("Given a fake archive that can't run the search", t, func() {
		server := fakeArchive(`{"error": "[UNSUPPORTED_VALUE] A requested parameter has an inappropriate value"}`, nil, nil)
		Reset(server.Close)
		client := internetarchive.NewClientWithBaseURL(server.URL)
		Convey("When finding an ebook", func() {
			_, err := client.FindEbook([]string{"prideprejudice00aust"})
			Convey("Then it returns ErrUpstreamUnavailable rather than no ebook", func() {
				So(errors.Is(err, internetarchive.ErrUpstreamUnavailable), ShouldBeTrue)
			})
		})
	})
	Convey("Given a fake archive that is down", t, func() {
		server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			responseWriter.WriteHeader(http.StatusServiceUnavailable)
		}))
		Reset(server.Close)
		client := internetarchive.NewClientWithBaseURL(server.URL)
		Convey("When finding an ebook", func() {
			_, err := client.FindEbook([]string{"prideprejudice00aust"})
			Convey("Then it returns ErrUpstreamUnavailable", func() {
				So(errors.Is(err, internetarchive.ErrUpstreamUnavailable), ShouldBeTrue)
			})
		})
	})
}

func TestOpenEbook(t *testing.T) {
	Convey("Given a fake archive serving a file in a folder of its item", t, func() {
		var requestedPath string
		server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			requestedPath = request.URL.EscapedPath()
			if request.URL.Path != "/download/item/folder/my book.epub" {
				http.NotFound(responseWriter, request)
				return
			}
			_, _ = responseWriter.Write([]byte("epub contents"))
		}))
		Reset(server.Close)
		client := internetarchive.NewClientWithBaseURL(server.URL)
		Convey("When opening the ebook", func() {
			download, err := client.OpenEbook(context.Background(), internetarchive.Ebook{Identifier: "item", FileName: "folder/my book.epub", Format: internetarchive.FormatEPUB})
			Convey("Then it escapes each segment of the file's path", func() {
				So(requestedPath, ShouldEqual, "/download/item/folder/my%20book.epub")
			})
			Convey("Then it streams the file's contents", func() {
				So(err, ShouldBeNil)
				defer download.Body.Close()
				contents, err := io.ReadAll(download.Body)
				So(err, ShouldBeNil)
				So(string(contents), ShouldEqual, "epub contents")
				So(download.ContentLength, ShouldEqual, 13)
			})
		})
		Convey("When opening a missing ebook", func() {
			_, err := client.OpenEbook(context.Background(), internetarchive.Ebook{Identifier: "item", FileName: "missing.epub"})
			Convey("Then it returns ErrUpstreamUnavailable", func() {
				So(errors.Is(err, internetarchive.ErrUpstreamUnavailable), ShouldBeTrue)
			})
		})
	})
}
