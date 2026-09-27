package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
)

func (api *API) SyncDevice(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	vars := mux.Vars(request)
	documents, err := api.syncer.SyncDevice(vars["id"])
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, documents)
}
