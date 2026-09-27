package httpapi

import (
	"net/http"

	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

func (api *API) CreateDevice(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	body, err := httpx.DecodeJSONBody[models.Device](request)
	if err != nil {
		return
	}
	device, err := api.store.CreateDevice(body)
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusCreated, device)
}
