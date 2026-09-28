package httpapi

import (
	"net/http"

	"github.com/kduong-dev/goutil/httpx"
)

func (api *API) ListDevices(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	devices, err := api.store.Devices.List()
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, devices)
}
