package httpapi

import (
	"net/http"

	"github.com/kduong-dev/goutil/httpx"
)

func (api *API) ListDevices(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	devices, err := api.deviceStore.List()
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, devices)
}
