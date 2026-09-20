package api

import (
	"net/http"

	"github.com/kqvd/reMarkableShelf/backend/internal/httpx"
)

func (handler *Handler) ListBooks(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	books, err := handler.Store.ListBooks()
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, books)
}
