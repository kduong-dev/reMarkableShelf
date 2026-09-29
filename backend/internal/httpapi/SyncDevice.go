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
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	vars := mux.Vars(request)
	// A sync the user asks for may restart the tablet's reading app to show
	// books copied to it.
	documents, err := api.syncer.SyncDeviceNow(vars["id"])
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, documents)
}
