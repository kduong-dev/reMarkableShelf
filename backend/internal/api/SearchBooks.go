package api

import (
	"net/http"

	"github.com/kduong-dev/goutil/httpx"
)

func (handler *Handler) SearchBooks(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	query := request.URL.Query().Get("q")
	results, err := handler.GoogleBooks.Search(query)
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, results)
}
