package api

import (
	"net/http"

	"github.com/kduong-dev/goutil/httpx"
	"github.com/kqvd/reMarkableShelf/backend/internal/models"
)

func (handler *Handler) CreateBook(responseWriter http.ResponseWriter, request *http.Request) {
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
	book, err := handler.Store.CreateBook(body)
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusCreated, book)
}
