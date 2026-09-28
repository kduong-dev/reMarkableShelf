package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
)

func (api *API) UnlinkDocument(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	vars := mux.Vars(request)
	err = api.store.Documents.Unlink(vars["id"], vars["uuid"])
	if err != nil {
		return
	}
	responseWriter.WriteHeader(http.StatusNoContent)
}
