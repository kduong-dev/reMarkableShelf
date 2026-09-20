package api

import (
	"net/http"

	"github.com/kqvd/reMarkableShelf/backend/internal/httpx"
	"github.com/kqvd/reMarkableShelf/backend/internal/models"
)

func (handler *Handler) CreateDevice(responseWriter http.ResponseWriter, request *http.Request) {
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
	device, err := handler.Store.CreateDevice(body)
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusCreated, device)
}
