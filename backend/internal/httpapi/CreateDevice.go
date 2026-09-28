package httpapi

import (
	"errors"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicesync"
)

// registerDeviceRequest carries the tablet's password alongside the device,
// used once to pair it and never stored or returned.
type registerDeviceRequest struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Password string `json:"password"`
}

func (api *API) CreateDevice(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			sendErrorResponse(responseWriter, err)
		}
	}()
	body, err := httpx.DecodeJSONBody[registerDeviceRequest](request)
	if err != nil {
		return
	}
	device, err := api.syncer.RegisterDevice(devicesync.RegisterDeviceInput{
		Name:     body.Name,
		Host:     body.Host,
		Password: body.Password,
	})
	var alreadyRegistered devicesync.DeviceAlreadyRegisteredError
	if errors.As(err, &alreadyRegistered) {
		err = merry.Wrap(err).WithHTTPCode(http.StatusConflict).
			WithUserMessagef("%s is already registered as %q; pair that one again instead of adding it twice", alreadyRegistered.Host, alreadyRegistered.Name)
	}
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusCreated, device)
}
