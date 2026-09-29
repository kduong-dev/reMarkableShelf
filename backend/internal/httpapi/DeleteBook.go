package httpapi

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookfilestore"
)

func (api *API) DeleteBook(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	vars := mux.Vars(request)
	bookID := vars["id"]
	// Deleting the book would forget its file's record but leave the file
	// in the storage service, so the file goes first.
	if err = api.bookFileStore.Delete(request.Context(), bookID); err != nil && !errors.Is(err, bookfilestore.ErrFileNotFound) {
		return
	}
	err = api.bookStore.Delete(bookID)
	if err != nil {
		return
	}
	responseWriter.WriteHeader(http.StatusNoContent)
}
