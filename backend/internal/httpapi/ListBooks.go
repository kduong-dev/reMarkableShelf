package httpapi

import (
	"net/http"

	"github.com/kduong-dev/goutil/httpx"
)

func (api *API) ListBooks(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	books, err := api.bookStore.List()
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, books)
}
