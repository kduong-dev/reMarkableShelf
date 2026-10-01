package httpapi

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

// ListSources lists the ebook sources in the order they're tried.
func (api *API) ListSources(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	sources, err := api.sourceStore.List()
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, sources)
}

type addSourceBody struct {
	Kind models.EbookSourceKind `json:"kind"`
	URL  string                 `json:"url"`
}

// AddSource installs a plugin or OPDS catalog, last in line, once it's
// answered at its URL; it's named by its own manifest or feed.
func (api *API) AddSource(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	body, err := httpx.DecodeJSONBody[addSourceBody](request)
	if err != nil {
		return
	}
	inspected, err := api.ebookSources.Inspect(request.Context(), body.Kind, strings.TrimSpace(body.URL))
	if err != nil {
		return
	}
	source, err := api.sourceStore.Create(inspected)
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusCreated, source)
}

type updateSourceBody struct {
	Enabled bool `json:"enabled"`
}

// UpdateSource enables or disables a source.
func (api *API) UpdateSource(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	body, err := httpx.DecodeJSONBody[updateSourceBody](request)
	if err != nil {
		return
	}
	source, err := api.sourceStore.SetEnabled(mux.Vars(request)["id"], body.Enabled)
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, source)
}

type reorderSourcesBody struct {
	IDs []string `json:"ids"`
}

// ReorderSources sets the order sources are tried in.
func (api *API) ReorderSources(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	body, err := httpx.DecodeJSONBody[reorderSourcesBody](request)
	if err != nil {
		return
	}
	if err = api.sourceStore.Reorder(body.IDs); err != nil {
		return
	}
	sources, err := api.sourceStore.List()
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, sources)
}

// DeleteSource uninstalls a source; files already fetched from it stay.
func (api *API) DeleteSource(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	if err = api.sourceStore.Delete(mux.Vars(request)["id"]); err != nil {
		return
	}
	responseWriter.WriteHeader(http.StatusNoContent)
}
