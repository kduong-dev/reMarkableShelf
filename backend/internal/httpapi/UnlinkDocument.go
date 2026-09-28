package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (api *API) UnlinkDocument(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	vars := mux.Vars(request)
	err = api.documentStore.Unlink(vars["id"], vars["uuid"])
	if err != nil {
		return
	}
	responseWriter.WriteHeader(http.StatusNoContent)
}
