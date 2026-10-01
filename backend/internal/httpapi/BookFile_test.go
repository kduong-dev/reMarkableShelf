package httpapi_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/httpapi"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/internetarchive"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/storetest"
)

func TestBookFile(t *testing.T) {
	Convey("Given the API with a book matched to a public domain work", t, func() {
		stores := storetest.OpenStores(t)
		book, err := stores.BookStore.Create(models.Book{Title: "Pride and Prejudice", OpenLibraryID: "OL66554W"})
		So(err, ShouldBeNil)
		handler := httpapi.NewHandler(httpapi.NewHandlerInput{
			BookFileStore: stores.BookFileStore,
			BookStore:     stores.BookStore,
			DeviceStore:   stores.DeviceStore,
			DocumentStore: stores.DocumentStore,
			OpenLibrary:   fakeOpenLibrary{scans: openlibrary.Scans{Title: "Pride and Prejudice", ArchiveIDs: []string{"pride42671gut"}}},
			InternetArchive: fakeInternetArchive{
				ebook:    internetarchive.Ebook{Identifier: "pride42671gut", FileName: "pg42671.pdf", Format: internetarchive.FormatPDF},
				contents: "%PDF-1.7 from the archive",
			},
			SourceStore: stores.SourceStore,
		})
		send := func(method, path string, body io.Reader) *httptest.ResponseRecorder {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(method, path, body))
			return recorder
		}
		fileOf := func(recorder *httptest.ResponseRecorder) models.BookFile {
			var file models.BookFile
			So(json.Unmarshal(recorder.Body.Bytes(), &file), ShouldBeNil)
			return file
		}
		filePath := "/api/books/" + book.ID + "/file"
		Convey("When the book has no file", func() {
			recorder := send(http.MethodGet, filePath, nil)
			Convey("Then getting it responds 404", func() {
				So(recorder.Code, ShouldEqual, http.StatusNotFound)
				So(recorder.Body.String(), ShouldContainSubstring, "no file saved on the server")
			})
		})
		Convey("When uploading a PDF", func() {
			recorder := send(http.MethodPut, filePath, strings.NewReader("%PDF-1.7 my own copy"))
			Convey("Then it's saved as an upload", func() {
				So(recorder.Code, ShouldEqual, http.StatusOK)
				file := fileOf(recorder)
				So(file.Format, ShouldEqual, models.FileTypePDF)
				So(file.Source, ShouldEqual, models.BookFileSourceUpload)
			})
			Convey("And downloading it", func() {
				recorder := send(http.MethodGet, filePath+"/content", nil)
				Convey("Then it's sent named after the book", func() {
					So(recorder.Code, ShouldEqual, http.StatusOK)
					So(recorder.Body.String(), ShouldEqual, "%PDF-1.7 my own copy")
					So(recorder.Header().Get("Content-Type"), ShouldEqual, "application/pdf")
					So(recorder.Header().Get("Content-Disposition"), ShouldEqual, `attachment; filename="Pride and Prejudice.pdf"`)
				})
			})
			Convey("And deleting it", func() {
				recorder := send(http.MethodDelete, filePath, nil)
				Convey("Then it's gone", func() {
					So(recorder.Code, ShouldEqual, http.StatusNoContent)
					So(send(http.MethodGet, filePath, nil).Code, ShouldEqual, http.StatusNotFound)
					So(stores.StorageService.Files, ShouldBeEmpty)
				})
			})
			Convey("And deleting the book", func() {
				recorder := send(http.MethodDelete, "/api/books/"+book.ID, nil)
				Convey("Then its file leaves the storage service too", func() {
					So(recorder.Code, ShouldEqual, http.StatusNoContent)
					So(stores.StorageService.Files, ShouldBeEmpty)
				})
			})
		})
		Convey("When uploading something that isn't an ebook", func() {
			recorder := send(http.MethodPut, filePath, strings.NewReader("just some notes"))
			Convey("Then it responds 415", func() {
				So(recorder.Code, ShouldEqual, http.StatusUnsupportedMediaType)
				So(recorder.Body.String(), ShouldContainSubstring, "only EPUB and PDF files")
			})
		})
		Convey("When uploading for a book that doesn't exist", func() {
			recorder := send(http.MethodPut, "/api/books/missing/file", strings.NewReader("%PDF-1.7"))
			Convey("Then it responds 404", func() {
				So(recorder.Code, ShouldEqual, http.StatusNotFound)
			})
		})
		Convey("When fetching the free ebook", func() {
			recorder := send(http.MethodPost, filePath+"/fetch", nil)
			Convey("Then the archive's file is saved as fetched from the Internet Archive", func() {
				So(recorder.Code, ShouldEqual, http.StatusOK)
				file := fileOf(recorder)
				So(file.Format, ShouldEqual, models.FileTypePDF)
				So(file.Source, ShouldEqual, models.BookFileSourceFetched)
				So(file.SourceName, ShouldEqual, "Internet Archive")
			})
		})
		Convey("When fetching for a book not matched to Open Library", func() {
			unmatched, err := stores.BookStore.Create(models.Book{Title: "Meeting notes"})
			So(err, ShouldBeNil)
			recorder := send(http.MethodPost, "/api/books/"+unmatched.ID+"/file/fetch", nil)
			Convey("Then it responds 404, as the Internet Archive needs an Open Library match and no other source is added", func() {
				So(recorder.Code, ShouldEqual, http.StatusNotFound)
				So(recorder.Body.String(), ShouldContainSubstring, "none of your ebook sources has this book")
			})
		})
	})
}
