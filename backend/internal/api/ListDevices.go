package api

import (
	"net/http"

	"github.com/kduong-dev/goutil/httpx"
)

func (handler *Handler) ListDevices(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	devices, err := handler.Store.ListDevices()
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, devices)
}
