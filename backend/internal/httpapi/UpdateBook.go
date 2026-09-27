package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

func (api *API) UpdateBook(responseWriter http.ResponseWriter, request *http.Request) {
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
	// Matching a book to an Open Library edition sets its ISBN; take that
	// edition's page count as adding a book does.
	api.applyEditionPageCount(&body)
	vars := mux.Vars(request)
	bookID := vars["id"]
	book, err := api.store.UpdateBook(bookID, body)
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, book)
}
