package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
)

func (handler *Handler) DeleteBook(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	vars := mux.Vars(request)
	bookID := vars["id"]
	err = handler.Store.DeleteBook(bookID)
	if err != nil {
		return
	}
	responseWriter.WriteHeader(http.StatusNoContent)
}
