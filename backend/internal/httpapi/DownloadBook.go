package httpapi

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/logx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/booksource"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

var contentTypes = map[string]string{
	"epub": "application/epub+zip",
	"pdf":  "application/pdf",
}

// DownloadBook streams an ebook of a search result, from the first enabled
// source that has one, named after the ebook. The result's title and
// author, passed as parameters, let sources other than the Internet
// Archive look for it.
func (api *API) DownloadBook(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	query := booksource.Query{
		Title:         request.URL.Query().Get("title"),
		Author:        request.URL.Query().Get("author"),
		OpenLibraryID: mux.Vars(request)["openLibraryId"],
	}
	_, ebook, download, err := api.openEbook(request.Context(), query, ebookChoice{})
	if err != nil {
		return
	}
	defer download.Body.Close()
	title := ebook.Title
	if title == "" {
		title = query.Title
	}
	sendAttachment(responseWriter, sendAttachmentInput{
		Title:         title,
		Format:        string(ebook.Format),
		ContentLength: download.ContentLength,
		Body:          download.Body,
	})
}

// sendAttachmentInput is an ebook to send, named after Title, with
// ContentLength -1 when unknown.
type sendAttachmentInput struct {
	Title         string
	Format        string
	ContentLength int64
	Body          io.Reader
}

// sendAttachment streams an ebook for the browser to save.
func sendAttachment(responseWriter http.ResponseWriter, input sendAttachmentInput) {
	fileName := fmt.Sprintf("%s.%s", fileNameOf(input.Title), input.Format)
	responseWriter.Header().Set("Content-Type", contentTypes[input.Format])
	responseWriter.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": fileName}))
	if input.ContentLength >= 0 {
		responseWriter.Header().Set("Content-Length", fmt.Sprint(input.ContentLength))
	}
	// Once the body has started, a failure can no longer be sent as an
	// error response.
	if _, err := io.Copy(responseWriter, input.Body); err != nil {
		logx.Warnf("sending %s: %v", fileName, err)
	}
}

// ebookChoice is an ebook the user picked from a source; left empty, the
// sources are asked in order.
type ebookChoice struct {
	SourceID string `json:"sourceId"`
	EbookID  string `json:"ebookId"`
}

// openEbook starts fetching the chosen ebook, or the preferred ebook of the
// first enabled source that has the book, and returns where it came from.
func (api *API) openEbook(ctx context.Context, query booksource.Query, choice ebookChoice) (models.EbookSource, booksource.Ebook, booksource.Download, error) {
	var source models.EbookSource
	var ebook booksource.Ebook
	var err error
	if choice.SourceID != "" {
		source, err = api.sourceStore.Get(choice.SourceID)
		ebook = booksource.Ebook{ID: choice.EbookID, Title: query.Title}
	} else {
		var sources []models.EbookSource
		if sources, err = api.sourceStore.List(); err == nil {
			source, ebook, err = api.ebookSources.FindFirst(ctx, sources, query)
		}
	}
	if err != nil {
		return models.EbookSource{}, booksource.Ebook{}, booksource.Download{}, err
	}
	opened, err := api.ebookSources.Open(source)
	if err != nil {
		return models.EbookSource{}, booksource.Ebook{}, booksource.Download{}, err
	}
	download, err := opened.OpenEbook(ctx, ebook.ID)
	if err != nil {
		return models.EbookSource{}, booksource.Ebook{}, booksource.Download{}, err
	}
	return source, ebook, download, nil
}

// queryOf is what to ask ebook sources for to find a book.
func queryOf(book models.Book) booksource.Query {
	return booksource.Query{Title: book.Title, Author: book.Author, ISBN: book.ISBN, OpenLibraryID: book.OpenLibraryID}
}

// fileNameOf keeps a title to the characters safe in a file name.
func fileNameOf(title string) string {
	name := strings.Map(func(character rune) rune {
		if unicode.IsControl(character) || strings.ContainsRune(`/\:*?"<>|`, character) {
			return -1
		}
		return character
	}, title)
	name = strings.TrimSpace(name)
	if name == "" {
		return "book"
	}
	return name
}
