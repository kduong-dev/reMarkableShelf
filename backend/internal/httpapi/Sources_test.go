package httpapi_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/foldersource"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/httpapi"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/storetest"
	"github.com/kduong-dev/reMarkableShelf/backend/pkg/sourceplugin"
)

func TestEbookSources(t *testing.T) {
	Convey("Given the API and a folder plugin holding a PDF of a book on the shelf", t, func() {
		directory := t.TempDir()
		So(os.WriteFile(filepath.Join(directory, "Clean Code.pdf"), []byte("%PDF-1.7 clean code"), 0o644), ShouldBeNil)
		plugin := httptest.NewServer(sourceplugin.NewHandler(foldersource.New("Purchases", directory)))
		Reset(plugin.Close)
		stores := storetest.OpenStores(t)
		book, err := stores.BookStore.Create(models.Book{Title: "Clean Code: a handbook of agile software craftsmanship"})
		So(err, ShouldBeNil)
		handler := httpapi.NewHandler(httpapi.NewHandlerInput{
			BookFileStore: stores.BookFileStore,
			BookStore:     stores.BookStore,
			DeviceStore:   stores.DeviceStore,
			DocumentStore: stores.DocumentStore,
			OpenLibrary:   fakeOpenLibrary{err: nil},
			SourceStore:   stores.SourceStore,
		})
		send := func(method, path, body string) *httptest.ResponseRecorder {
			recorder := httptest.NewRecorder()
			var reader io.Reader
			if body != "" {
				reader = strings.NewReader(body)
			}
			handler.ServeHTTP(recorder, httptest.NewRequest(method, path, reader))
			return recorder
		}
		decode := func(recorder *httptest.ResponseRecorder, output any) {
			So(json.Unmarshal(recorder.Body.Bytes(), output), ShouldBeNil)
		}
		Convey("When adding the plugin by its URL", func() {
			recorder := send(http.MethodPost, "/api/sources", `{"kind": "plugin", "url": "`+plugin.URL+`"}`)
			var added models.EbookSource
			decode(recorder, &added)
			Convey("Then it's added, named by its manifest, after the Internet Archive", func() {
				So(recorder.Code, ShouldEqual, http.StatusCreated)
				So(added.Name, ShouldEqual, "Purchases")
				var sources []models.EbookSource
				decode(send(http.MethodGet, "/api/sources", ""), &sources)
				So(sources, ShouldHaveLength, 2)
				So(sources[0].ID, ShouldEqual, models.BuiltInEbookSourceID)
				So(sources[1].ID, ShouldEqual, added.ID)
			})
			Convey("And listing the book's ebooks", func() {
				var listed []struct {
					Source models.EbookSource `json:"source"`
					Ebooks []struct {
						ID     string `json:"id"`
						Format string `json:"format"`
					} `json:"ebooks"`
				}
				recorder := send(http.MethodGet, "/api/books/"+book.ID+"/ebooks", "")
				decode(recorder, &listed)
				Convey("Then every enabled source answers, in order, the plugin with the PDF", func() {
					So(recorder.Code, ShouldEqual, http.StatusOK)
					So(listed, ShouldHaveLength, 2)
					So(listed[0].Ebooks, ShouldBeEmpty)
					So(listed[1].Source.Name, ShouldEqual, "Purchases")
					So(listed[1].Ebooks, ShouldHaveLength, 1)
					So(listed[1].Ebooks[0].ID, ShouldEqual, "Clean Code.pdf")
				})
				Convey("And fetching that ebook", func() {
					recorder := send(http.MethodPost, "/api/books/"+book.ID+"/file/fetch", `{"sourceId": "`+added.ID+`", "ebookId": "Clean Code.pdf"}`)
					var file models.BookFile
					decode(recorder, &file)
					Convey("Then it's saved, named as from the plugin", func() {
						So(recorder.Code, ShouldEqual, http.StatusOK)
						So(file.Format, ShouldEqual, models.FileTypePDF)
						So(file.Source, ShouldEqual, models.BookFileSourceFetched)
						So(file.SourceName, ShouldEqual, "Purchases")
					})
				})
			})
			Convey("And fetching without choosing, with the plugin moved first", func() {
				So(send(http.MethodPut, "/api/sources/order", `{"ids": ["`+added.ID+`", "`+models.BuiltInEbookSourceID+`"]}`).Code, ShouldEqual, http.StatusOK)
				recorder := send(http.MethodPost, "/api/books/"+book.ID+"/file/fetch", "")
				var file models.BookFile
				decode(recorder, &file)
				Convey("Then the first source that has the book supplies it", func() {
					So(recorder.Code, ShouldEqual, http.StatusOK)
					So(file.SourceName, ShouldEqual, "Purchases")
				})
			})
			Convey("And disabling it before fetching", func() {
				So(send(http.MethodPut, "/api/sources/"+added.ID, `{"enabled": false}`).Code, ShouldEqual, http.StatusOK)
				recorder := send(http.MethodPost, "/api/books/"+book.ID+"/file/fetch", "")
				Convey("Then no enabled source has it", func() {
					So(recorder.Code, ShouldEqual, http.StatusNotFound)
				})
			})
		})
		Convey("When adding a URL where no plugin answers", func() {
			recorder := send(http.MethodPost, "/api/sources", `{"kind": "plugin", "url": "`+plugin.URL+`/nothing-here"}`)
			Convey("Then it's refused", func() {
				So(recorder.Code, ShouldEqual, http.StatusBadRequest)
				So(recorder.Body.String(), ShouldContainSubstring, "no ebook source plugin answered")
			})
		})
		Convey("When removing the built-in source", func() {
			recorder := send(http.MethodDelete, "/api/sources/"+models.BuiltInEbookSourceID, "")
			Convey("Then it's refused", func() {
				So(recorder.Code, ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}
