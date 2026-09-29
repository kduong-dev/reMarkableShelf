package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
)

// MarkNotABook marks a tablet document as not a book, such as a planner
// template, unlinking it and keeping sync from linking it again.
func (api *API) MarkNotABook(responseWriter http.ResponseWriter, request *http.Request) {
	api.setNotABook(responseWriter, request, true)
}

// UnmarkNotABook clears the mark, so the document can be linked again.
func (api *API) UnmarkNotABook(responseWriter http.ResponseWriter, request *http.Request) {
	api.setNotABook(responseWriter, request, false)
}

func (api *API) setNotABook(responseWriter http.ResponseWriter, request *http.Request, notABook bool) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	vars := mux.Vars(request)
	err = api.documentStore.SetNotABook(vars["id"], vars["uuid"], notABook)
	if err != nil {
		return
	}
	responseWriter.WriteHeader(http.StatusNoContent)
}
