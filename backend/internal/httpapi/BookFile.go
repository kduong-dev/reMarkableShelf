package httpapi

import (
	"errors"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookfilestore"
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

// FetchBookFile saves the free ebook of a public domain book, looked up by
// the Open Library work it's matched to.
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
	if book.OpenLibraryID == "" {
		err = merry.Here(ErrNoOpenLibraryWork)
		return
	}
	_, _, download, err := api.openPublicEbook(request.Context(), book.OpenLibraryID)
	if err != nil {
		return
	}
	defer download.Body.Close()
	file, err := api.bookFileStore.Save(request.Context(), bookfilestore.SaveInput{
		BookID: book.ID,
		Source: models.BookFileSourceOpenLibrary,
		Body:   download.Body,
	})
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, file)
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
