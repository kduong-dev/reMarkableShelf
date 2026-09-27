package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
)

type pairDeviceRequest struct {
	Password string `json:"password"`
}

func (api *API) PairDevice(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	body, err := httpx.DecodeJSONBody[pairDeviceRequest](request)
	if err != nil {
		return
	}
	device, err := api.syncer.PairDevice(mux.Vars(request)["id"], body.Password)
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, device)
}
