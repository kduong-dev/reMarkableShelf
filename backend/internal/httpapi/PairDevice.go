package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicesync"
)

type pairDeviceRequest struct {
	Password          string `json:"password"`
	AcceptNewIdentity bool   `json:"acceptNewIdentity"`
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
	device, err := api.syncer.PairDevice(devicesync.PairDeviceInput{
		DeviceID:          mux.Vars(request)["id"],
		Password:          body.Password,
		AcceptNewIdentity: body.AcceptNewIdentity,
	})
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, device)
}
