package sourceplugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

var contentTypes = map[Format]string{
	FormatEPUB: "application/epub+zip",
	FormatPDF:  "application/pdf",
}

// NewHandler serves source by the plugin contract.
func NewHandler(source Source) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /manifest", func(responseWriter http.ResponseWriter, request *http.Request) {
		sendJSON(responseWriter, source.Manifest())
	})
	mux.HandleFunc("POST /ebooks/find", func(responseWriter http.ResponseWriter, request *http.Request) {
		var find FindRequest
		if err := json.NewDecoder(request.Body).Decode(&find); err != nil {
			http.Error(responseWriter, "invalid find request", http.StatusBadRequest)
			return
		}
		ebooks, err := source.FindEbooks(request.Context(), find)
		if err != nil {
			log.Printf("finding ebooks of %q: %v", find.Title, err)
			http.Error(responseWriter, "finding ebooks failed", http.StatusBadGateway)
			return
		}
		if ebooks == nil {
			ebooks = []Ebook{}
		}
		sendJSON(responseWriter, FindResponse{Ebooks: ebooks})
	})
	mux.HandleFunc("GET /ebooks/{id}/content", func(responseWriter http.ResponseWriter, request *http.Request) {
		content, err := source.OpenEbook(request.Context(), request.PathValue("id"))
		if errors.Is(err, ErrEbookNotFound) {
			http.NotFound(responseWriter, request)
			return
		}
		if err != nil {
			log.Printf("opening ebook %q: %v", request.PathValue("id"), err)
			http.Error(responseWriter, "opening the ebook failed", http.StatusBadGateway)
			return
		}
		defer content.Body.Close()
		responseWriter.Header().Set("Content-Type", contentTypes[content.Format])
		if content.Size > 0 {
			responseWriter.Header().Set("Content-Length", fmt.Sprint(content.Size))
		}
		_, _ = io.Copy(responseWriter, content.Body)
	})
	return mux
}

func sendJSON(responseWriter http.ResponseWriter, body any) {
	responseWriter.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(responseWriter).Encode(body)
}
