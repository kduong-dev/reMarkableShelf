package booksource_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/booksource"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/pkg/sourceplugin"
)

// fakePlugin is a plugin with one ebook of each title in files.
type fakePlugin struct {
	files    map[string]string
	requests *[]sourceplugin.FindRequest
}

func (plugin fakePlugin) Manifest() sourceplugin.Manifest {
	return sourceplugin.Manifest{Name: "Bookshelf folder"}
}

func (plugin fakePlugin) FindEbooks(ctx context.Context, request sourceplugin.FindRequest) ([]sourceplugin.Ebook, error) {
	if plugin.requests != nil {
		*plugin.requests = append(*plugin.requests, request)
	}
	if _, ok := plugin.files[request.Title]; !ok {
		return nil, nil
	}
	return []sourceplugin.Ebook{
		{ID: "notes/" + request.Title + ".txt", Title: request.Title, Format: "txt"},
		{ID: "books/" + request.Title + ".epub", Title: request.Title, Format: sourceplugin.FormatEPUB, Size: 12},
	}, nil
}

func (plugin fakePlugin) OpenEbook(ctx context.Context, id string) (sourceplugin.Content, error) {
	for title, contents := range plugin.files {
		if id == "books/"+title+".epub" {
			return sourceplugin.Content{Body: io.NopCloser(strings.NewReader(contents)), Format: sourceplugin.FormatEPUB, Size: int64(len(contents))}, nil
		}
	}
	return sourceplugin.Content{}, sourceplugin.ErrEbookNotFound
}

func TestPluginSource(t *testing.T) {
	Convey("Given a plugin serving one book", t, func() {
		var requests []sourceplugin.FindRequest
		server := httptest.NewServer(sourceplugin.NewHandler(fakePlugin{files: map[string]string{"Emma": "emma epub"}, requests: &requests}))
		Reset(server.Close)
		factory := booksource.NewFactory(booksource.NewFactoryInput{})
		ctx := context.Background()
		Convey("When adding it by its URL", func() {
			source, err := factory.Inspect(ctx, models.EbookSourceKindPlugin, server.URL+"/")
			Convey("Then it's named by its manifest", func() {
				So(err, ShouldBeNil)
				So(source.Name, ShouldEqual, "Bookshelf folder")
				So(source.URL, ShouldEqual, server.URL)
			})
			Convey("And finding the book", func() {
				opened, err := factory.Open(source)
				So(err, ShouldBeNil)
				ebooks, err := opened.FindEbooks(ctx, booksource.Query{Title: "Emma", Author: "Jane Austen", OpenLibraryID: "OL66554W"})
				Convey("Then it passes the book along and keeps only EPUBs and PDFs", func() {
					So(err, ShouldBeNil)
					So(requests, ShouldResemble, []sourceplugin.FindRequest{{Title: "Emma", Author: "Jane Austen", OpenLibraryID: "OL66554W"}})
					So(ebooks, ShouldResemble, []booksource.Ebook{{ID: "books/Emma.epub", Title: "Emma", Format: models.FileTypeEPUB, Size: 12}})
				})
				Convey("Then the ebook can be fetched by its ID, slashes and all", func() {
					download, err := opened.OpenEbook(ctx, ebooks[0].ID)
					So(err, ShouldBeNil)
					defer download.Body.Close()
					contents, _ := io.ReadAll(download.Body)
					So(string(contents), ShouldEqual, "emma epub")
				})
				Convey("Then an ebook it no longer has is ErrEbookNotFound", func() {
					_, err := opened.OpenEbook(ctx, "books/Persuasion.epub")
					So(errors.Is(err, booksource.ErrEbookNotFound), ShouldBeTrue)
				})
			})
		})
	})
	Convey("Given a URL where no plugin answers", t, func() {
		server := httptest.NewServer(http.NotFoundHandler())
		Reset(server.Close)
		factory := booksource.NewFactory(booksource.NewFactoryInput{})
		Convey("When adding it", func() {
			_, err := factory.Inspect(context.Background(), models.EbookSourceKindPlugin, server.URL)
			Convey("Then it's refused as not a plugin", func() {
				So(errors.Is(err, booksource.ErrNotPlugin), ShouldBeTrue)
			})
		})
		Convey("When adding a URL that isn't on the web", func() {
			_, err := factory.Inspect(context.Background(), models.EbookSourceKindPlugin, "file:///etc/passwd")
			Convey("Then it's refused", func() {
				So(errors.Is(err, booksource.ErrInvalidURL), ShouldBeTrue)
			})
		})
	})
}

// fakeCatalog serves an OPDS catalog shaped like Project Gutenberg's: a
// root feed linking an OpenSearch description, search results linking each
// book's own feed, and that feed listing its files.
func fakeCatalog(t *testing.T) *httptest.Server {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		write := func(body string) { _, _ = responseWriter.Write([]byte(strings.ReplaceAll(body, "SERVER", server.URL))) }
		switch request.URL.Path {
		case "/opds":
			write(`<feed xmlns="http://www.w3.org/2005/Atom"><title>Classics Catalog</title>
				<link rel="search" type="application/opensearchdescription+xml" href="/osd.xml"/></feed>`)
		case "/osd.xml":
			write(`<OpenSearchDescription xmlns="http://a9.com/-/spec/opensearch/1.1/">
				<Url type="text/html" template="SERVER/html?q={searchTerms}"/>
				<Url type="application/atom+xml" template="SERVER/search?q={searchTerms}&amp;page={startPage?}"/></OpenSearchDescription>`)
		case "/search":
			if request.URL.Query().Get("q") != "Frankenstein Mary Shelley" {
				write(`<feed xmlns="http://www.w3.org/2005/Atom"><title>No results</title></feed>`)
				return
			}
			write(`<feed xmlns="http://www.w3.org/2005/Atom"><title>Results</title>
				<entry><title>Frankenstein; or, the Modern Prometheus</title>
					<link rel="subsection" type="application/atom+xml;profile=opds-catalog" href="/books/84"/></entry>
				<entry><title>Frankenstein Unbound</title>
					<link rel="http://opds-spec.org/acquisition" type="application/epub+zip" href="/files/unbound.epub"/></entry>
				<entry><title>The Bride of Frankenstein</title>
					<link rel="http://opds-spec.org/acquisition" type="application/epub+zip" href="/files/bride.epub"/></entry></feed>`)
		case "/books/84":
			write(`<feed xmlns="http://www.w3.org/2005/Atom"><title>Frankenstein</title>
				<entry><title>Frankenstein; or, the Modern Prometheus</title><author><name>Shelley, Mary Wollstonecraft</name></author>
					<link rel="http://opds-spec.org/acquisition" type="application/epub+zip" title="EPUB (no images)" length="356059" href="/files/84.noimages.epub"/>
					<link rel="http://opds-spec.org/acquisition" type="application/epub+zip" title="EPUB3 (with images)" length="474161" href="/files/84.images.epub"/>
					<link rel="http://opds-spec.org/acquisition" type="application/x-mobipocket-ebook" href="/files/84.mobi"/>
					<link rel="related" type="application/atom+xml" href="/authors/68"/></entry></feed>`)
		case "/files/84.images.epub":
			write("frankenstein epub")
		default:
			http.NotFound(responseWriter, request)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestOPDSSource(t *testing.T) {
	Convey("Given an OPDS catalog", t, func() {
		server := fakeCatalog(t)
		factory := booksource.NewFactory(booksource.NewFactoryInput{})
		ctx := context.Background()
		Convey("When adding it by its root feed", func() {
			source, err := factory.Inspect(ctx, models.EbookSourceKindOPDS, server.URL+"/opds")
			Convey("Then it's named by its feed and finds its search through OpenSearch", func() {
				So(err, ShouldBeNil)
				So(source.Name, ShouldEqual, "Classics Catalog")
				So(source.SearchURL, ShouldEqual, server.URL+"/search?q={searchTerms}&page={startPage?}")
			})
			Convey("And finding a book listed on its own page", func() {
				opened, err := factory.Open(source)
				So(err, ShouldBeNil)
				ebooks, err := opened.FindEbooks(ctx, booksource.Query{Title: "Frankenstein", Author: "Mary Shelley"})
				Convey("Then it follows the book's page to its EPUBs, leaving out other books and formats", func() {
					So(err, ShouldBeNil)
					So(ebooks, ShouldResemble, []booksource.Ebook{
						{ID: server.URL + "/files/84.noimages.epub", Title: "Frankenstein; or, the Modern Prometheus", Author: "Mary Wollstonecraft Shelley", Format: models.FileTypeEPUB, Size: 356059, Description: "EPUB (no images)"},
						{ID: server.URL + "/files/84.images.epub", Title: "Frankenstein; or, the Modern Prometheus", Author: "Mary Wollstonecraft Shelley", Format: models.FileTypeEPUB, Size: 474161, Description: "EPUB3 (with images)"},
					})
				})
				Convey("Then the edition with images is preferred", func() {
					So(booksource.Preferred(ebooks).ID, ShouldEqual, server.URL+"/files/84.images.epub")
				})
				Convey("Then it can be fetched", func() {
					download, err := opened.OpenEbook(ctx, booksource.Preferred(ebooks).ID)
					So(err, ShouldBeNil)
					defer download.Body.Close()
					contents, _ := io.ReadAll(download.Body)
					So(string(contents), ShouldEqual, "frankenstein epub")
				})
			})
			Convey("And finding a book it doesn't have", func() {
				opened, err := factory.Open(source)
				So(err, ShouldBeNil)
				ebooks, err := opened.FindEbooks(ctx, booksource.Query{Title: "Dune", Author: "Frank Herbert"})
				Convey("Then it finds none", func() {
					So(err, ShouldBeNil)
					So(ebooks, ShouldBeEmpty)
				})
			})
		})
		Convey("When adding a feed with no search", func() {
			_, err := factory.Inspect(ctx, models.EbookSourceKindOPDS, server.URL+"/books/84")
			Convey("Then it's refused as not searchable", func() {
				So(errors.Is(err, booksource.ErrNotSearchable), ShouldBeTrue)
			})
		})
		Convey("When adding something that isn't a feed", func() {
			_, err := factory.Inspect(ctx, models.EbookSourceKindOPDS, server.URL+"/files/84.images.epub")
			Convey("Then it's refused as not OPDS", func() {
				So(errors.Is(err, booksource.ErrNotOPDS), ShouldBeTrue)
			})
		})
	})
}

func TestFindFirst(t *testing.T) {
	Convey("Given a plugin with Emma, a plugin that's down, and a disabled plugin with Persuasion", t, func() {
		withEmma := httptest.NewServer(sourceplugin.NewHandler(fakePlugin{files: map[string]string{"Emma": "emma"}}))
		Reset(withEmma.Close)
		down := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			responseWriter.WriteHeader(http.StatusServiceUnavailable)
		}))
		Reset(down.Close)
		withPersuasion := httptest.NewServer(sourceplugin.NewHandler(fakePlugin{files: map[string]string{"Persuasion": "persuasion"}}))
		Reset(withPersuasion.Close)
		sources := []models.EbookSource{
			{ID: "down", Name: "Down", Kind: models.EbookSourceKindPlugin, URL: down.URL, Enabled: true},
			{ID: "emma", Name: "Has Emma", Kind: models.EbookSourceKindPlugin, URL: withEmma.URL, Enabled: true},
			{ID: "persuasion", Name: "Has Persuasion", Kind: models.EbookSourceKindPlugin, URL: withPersuasion.URL, Enabled: false},
		}
		factory := booksource.NewFactory(booksource.NewFactoryInput{})
		ctx := context.Background()
		Convey("When finding Emma", func() {
			source, ebook, err := factory.FindFirst(ctx, sources, booksource.Query{Title: "Emma"})
			Convey("Then it passes over the source that's down to the next that has it", func() {
				So(err, ShouldBeNil)
				So(source.ID, ShouldEqual, "emma")
				So(ebook.ID, ShouldEqual, "books/Emma.epub")
			})
		})
		Convey("When finding Persuasion", func() {
			_, _, err := factory.FindFirst(ctx, sources, booksource.Query{Title: "Persuasion"})
			Convey("Then the disabled source isn't asked, so no source has it", func() {
				So(errors.Is(err, booksource.ErrNoEbook), ShouldBeTrue)
			})
		})
		Convey("When the only enabled source is down", func() {
			_, _, err := factory.FindFirst(ctx, sources[:1], booksource.Query{Title: "Emma"})
			Convey("Then it reports the source unavailable rather than missing the book", func() {
				So(errors.Is(err, booksource.ErrSourceUnavailable), ShouldBeTrue)
			})
		})
		Convey("When finding Emma in every source at once", func() {
			found := factory.FindAll(ctx, sources, booksource.Query{Title: "Emma"})
			Convey("Then each enabled source answers in order, the one down with its error", func() {
				So(found, ShouldHaveLength, 2)
				So(found[0].Source.ID, ShouldEqual, "down")
				So(errors.Is(found[0].Err, booksource.ErrSourceUnavailable), ShouldBeTrue)
				So(found[1].Source.ID, ShouldEqual, "emma")
				So(found[1].Ebooks, ShouldHaveLength, 1)
			})
		})
	})
}

func TestOPDSRetries(t *testing.T) {
	Convey("Given a catalog whose search times out once, then answers", t, func() {
		Reset(booksource.SetRetryDelay(time.Millisecond))
		catalog := fakeCatalog(t)
		searches := 0
		flaky := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			if request.URL.Path == "/search" {
				searches++
				if searches == 1 {
					responseWriter.WriteHeader(http.StatusGatewayTimeout)
					return
				}
			}
			proxied, err := http.Get(catalog.URL + request.URL.RequestURI())
			if err != nil {
				http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
				return
			}
			defer proxied.Body.Close()
			responseWriter.WriteHeader(proxied.StatusCode)
			_, _ = io.Copy(responseWriter, proxied.Body)
		}))
		Reset(flaky.Close)
		factory := booksource.NewFactory(booksource.NewFactoryInput{})
		opened, err := factory.Open(models.EbookSource{Kind: models.EbookSourceKindOPDS, SearchURL: flaky.URL + "/search?q={searchTerms}"})
		So(err, ShouldBeNil)
		Convey("When finding a book", func() {
			ebooks, err := opened.FindEbooks(context.Background(), booksource.Query{Title: "Frankenstein", Author: "Mary Shelley"})
			Convey("Then it retries the search and finds it", func() {
				So(err, ShouldBeNil)
				So(searches, ShouldEqual, 2)
				So(ebooks, ShouldNotBeEmpty)
			})
		})
	})
}
