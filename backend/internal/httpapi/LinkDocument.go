package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
)

type linkDocumentRequest struct {
	BookID string `json:"bookId"`
}

func (api *API) LinkDocument(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	body, err := httpx.DecodeJSONBody[linkDocumentRequest](request)
	if err != nil {
		return
	}
	vars := mux.Vars(request)
	bookID := vars["id"]
	documentUUID := vars["uuid"]
	err = api.store.LinkDocumentToBook(bookID, documentUUID, body.BookID)
	if err != nil {
		return
	}
	responseWriter.WriteHeader(http.StatusNoContent)
}
