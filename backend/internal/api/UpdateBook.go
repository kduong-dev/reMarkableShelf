package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kqvd/reMarkableShelf/backend/internal/models"
)

func (handler *Handler) UpdateBook(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	body, err := httpx.DecodeJSONBody[models.Book](request)
	if err != nil {
		return
	}
	vars := mux.Vars(request)
	bookID := vars["id"]
	book, err := handler.Store.UpdateBook(bookID, body)
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, book)
}
