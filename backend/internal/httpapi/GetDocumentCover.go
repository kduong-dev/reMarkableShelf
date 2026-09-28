package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/gorilla/mux"
)

// GetDocumentCover serves a tablet document's synced cover. Its URL stays the
// same when the cover changes, so browsers revalidate it against an ETag of
// the image rather than caching it outright.
func (api *API) GetDocumentCover(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			sendErrorResponse(responseWriter, err)
		}
	}()
	vars := mux.Vars(request)
	image, err := api.store.Documents.GetCover(vars["id"], vars["uuid"])
	if err != nil {
		return
	}
	digest := sha256.Sum256(image)
	etag := `"` + hex.EncodeToString(digest[:8]) + `"`
	responseWriter.Header().Set("ETag", etag)
	responseWriter.Header().Set("Cache-Control", "no-cache")
	if request.Header.Get("If-None-Match") == etag {
		responseWriter.WriteHeader(http.StatusNotModified)
		return
	}
	responseWriter.Header().Set("Content-Type", "image/png")
	_, _ = responseWriter.Write(image)
}
