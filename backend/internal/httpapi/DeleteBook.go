package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
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
	err = api.store.Books.Delete(bookID)
	if err != nil {
		return
	}
	responseWriter.WriteHeader(http.StatusNoContent)
}
