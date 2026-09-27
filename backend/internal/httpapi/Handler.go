package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/remarkable"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/store"
)

type NewHandlerInput struct {
	Store            *store.Store
	OpenLibrary      openlibrary.API
	RemarkableConfig remarkable.Config
}

func NewHandler(input NewHandlerInput) http.Handler {
	api := &API{
		store:            input.Store,
		openLibrary:      input.OpenLibrary,
		remarkableConfig: input.RemarkableConfig,
	}
	router := mux.NewRouter().StrictSlash(true)
	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/books", api.ListBooks).Methods(http.MethodGet)
	apiRouter.HandleFunc("/books", api.CreateBook).Methods(http.MethodPost)
	apiRouter.HandleFunc("/books/{id}", api.GetBook).Methods(http.MethodGet)
	apiRouter.HandleFunc("/books/{id}", api.UpdateBook).Methods(http.MethodPut)
	apiRouter.HandleFunc("/books/{id}", api.DeleteBook).Methods(http.MethodDelete)
	apiRouter.HandleFunc("/search/books", api.SearchBooks).Methods(http.MethodGet)
	apiRouter.HandleFunc("/devices", api.ListDevices).Methods(http.MethodGet)
	apiRouter.HandleFunc("/devices", api.CreateDevice).Methods(http.MethodPost)
	apiRouter.HandleFunc("/devices/{id}/sync", api.SyncDevice).Methods(http.MethodPost)
	apiRouter.HandleFunc("/devices/{id}/documents", api.ListDocuments).Methods(http.MethodGet)
	apiRouter.HandleFunc("/devices/{id}/documents/{uuid}/link", api.LinkDocument).Methods(http.MethodPost)
	return httpx.HandlerWithCORS(router)
}
