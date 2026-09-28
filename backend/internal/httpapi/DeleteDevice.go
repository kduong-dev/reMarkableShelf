package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (api *API) DeleteDevice(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	err = api.deviceStore.Delete(mux.Vars(request)["id"])
	if err != nil {
		return
	}
	responseWriter.WriteHeader(http.StatusNoContent)
}
