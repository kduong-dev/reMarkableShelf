package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
)

type updateDeviceRequest struct {
	Name string `json:"name"`
}

func (api *API) UpdateDevice(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	body, err := httpx.DecodeJSONBody[updateDeviceRequest](request)
	if err != nil {
		return
	}
	device, err := api.deviceStore.Rename(mux.Vars(request)["id"], body.Name)
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, device)
}
