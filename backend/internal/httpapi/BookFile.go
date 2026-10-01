package httpapi

import (
	"errors"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/goutil/logx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookfilestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/booksource"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

// maximumUploadBytes caps an uploaded ebook; scanned PDFs run to tens of
// megabytes.
const maximumUploadBytes = 300 << 20

// GetBookFile describes the file saved for a book and the tablets it has
// been copied to.
func (api *API) GetBookFile(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	file, err := api.bookFileStore.Get(mux.Vars(request)["id"])
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, file)
}

// UploadBookFile saves the EPUB or PDF sent as the request body for a book,
// replacing any file it had.
func (api *API) UploadBookFile(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	book, err := api.bookStore.Get(mux.Vars(request)["id"])
	if err != nil {
		return
	}
	file, err := api.bookFileStore.Save(request.Context(), bookfilestore.SaveInput{
		BookID: book.ID,
		Source: models.BookFileSourceUpload,
		Body:   http.MaxBytesReader(responseWriter, request.Body, maximumUploadBytes),
	})
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		err = merry.Here(ErrFileTooLarge).WithUserMessagef("the file must be at most %d MB", maximumUploadBytes>>20)
	}
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, file)
}

// FetchBookFile saves an ebook of the book on the server: the one picked
// from a source, if the body names one, else the preferred ebook of the
// first enabled source that has the book.
func (api *API) FetchBookFile(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	book, err := api.bookStore.Get(mux.Vars(request)["id"])
	if err != nil {
		return
	}
	var choice ebookChoice
	if request.ContentLength != 0 {
		if choice, err = httpx.DecodeJSONBody[ebookChoice](request); err != nil {
			return
		}
	}
	source, _, download, err := api.openEbook(request.Context(), queryOf(book), choice)
	if err != nil {
		return
	}
	defer download.Body.Close()
	file, err := api.bookFileStore.Save(request.Context(), bookfilestore.SaveInput{
		BookID:     book.ID,
		Source:     models.BookFileSourceFetched,
		SourceName: source.Name,
		Body:       download.Body,
	})
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, file)
}

// sourceEbooks is what one source found of a book, with Error saying why it
// couldn't look.
type sourceEbooks struct {
	Source models.EbookSource `json:"source"`
	Ebooks []booksource.Ebook `json:"ebooks"`
	Error  string             `json:"error,omitempty"`
}

// ListBookEbooks asks every enabled source for ebooks of the book, so one
// can be picked, listing the sources in priority order.
func (api *API) ListBookEbooks(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	book, err := api.bookStore.Get(mux.Vars(request)["id"])
	if err != nil {
		return
	}
	sources, err := api.sourceStore.List()
	if err != nil {
		return
	}
	listed := []sourceEbooks{}
	for _, found := range api.ebookSources.FindAll(request.Context(), sources, queryOf(book)) {
		entry := sourceEbooks{Source: found.Source, Ebooks: found.Ebooks}
		if entry.Ebooks == nil {
			entry.Ebooks = []booksource.Ebook{}
		}
		if found.Err != nil {
			logx.Warnf("finding ebooks of %q in %s: %v", book.Title, found.Source.Name, found.Err)
			entry.Error = "couldn't search this source"
			if errors.Is(found.Err, booksource.ErrSourceUnavailable) {
				entry.Error = "the source didn't answer"
			}
		}
		listed = append(listed, entry)
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, listed)
}

// DownloadBookFile sends the file saved for a book, named after the book.
func (api *API) DownloadBookFile(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	book, err := api.bookStore.Get(mux.Vars(request)["id"])
	if err != nil {
		return
	}
	file, body, err := api.bookFileStore.Open(request.Context(), book.ID)
	if err != nil {
		return
	}
	defer body.Close()
	sendAttachment(responseWriter, sendAttachmentInput{
		Title:         book.Title,
		Format:        string(file.Format),
		ContentLength: file.Size,
		Body:          body,
	})
}

// DeleteBookFile removes the file saved for a book. Copies already on
// tablets stay there.
func (api *API) DeleteBookFile(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	err = api.bookFileStore.Delete(request.Context(), mux.Vars(request)["id"])
	if err != nil {
		return
	}
	responseWriter.WriteHeader(http.StatusNoContent)
}
