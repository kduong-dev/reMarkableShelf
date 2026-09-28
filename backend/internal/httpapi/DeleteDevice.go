package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
)

func (api *API) DeleteDevice(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	err = api.store.Devices.Delete(mux.Vars(request)["id"])
	if err != nil {
		return
	}
	responseWriter.WriteHeader(http.StatusNoContent)
}
