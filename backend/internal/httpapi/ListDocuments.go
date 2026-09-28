package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
)

func (api *API) ListDocuments(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			sendErrorResponse(responseWriter, err)
		}
	}()
	vars := mux.Vars(request)
	deviceID := vars["id"]
	documents, err := api.store.Documents.ListByDevice(deviceID)
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, documents)
}
