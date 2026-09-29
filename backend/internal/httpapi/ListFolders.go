package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
)

// ListFolders lists the folders in a device's library, as of its last sync.
func (api *API) ListFolders(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	folders, err := api.documentStore.ListFolders(mux.Vars(request)["id"])
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, folders)
}
