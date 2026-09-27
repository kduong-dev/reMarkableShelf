package httpapi

import (
	"net/http"

	"github.com/kduong-dev/goutil/httpx"
)

func (api *API) ListBooks(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	books, err := api.store.ListBooks()
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, books)
}
