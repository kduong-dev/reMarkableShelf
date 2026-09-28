package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicesync"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/documentstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
)

type NewHandlerInput struct {
	BookStore     bookstore.Store
	DeviceStore   devicestore.Store
	DocumentStore documentstore.Store
	OpenLibrary   openlibrary.API
	Syncer        *devicesync.Syncer
}

func NewHandler(input NewHandlerInput) http.Handler {
	api := &API{
		bookStore:     input.BookStore,
		deviceStore:   input.DeviceStore,
		documentStore: input.DocumentStore,
		openLibrary:   input.OpenLibrary,
		syncer:        input.Syncer,
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
	apiRouter.HandleFunc("/devices/{id}", api.UpdateDevice).Methods(http.MethodPut)
	apiRouter.HandleFunc("/devices/{id}", api.DeleteDevice).Methods(http.MethodDelete)
	apiRouter.HandleFunc("/devices/{id}/pair", api.PairDevice).Methods(http.MethodPost)
	apiRouter.HandleFunc("/devices/{id}/sync", api.SyncDevice).Methods(http.MethodPost)
	apiRouter.HandleFunc("/devices/{id}/documents", api.ListDocuments).Methods(http.MethodGet)
	apiRouter.HandleFunc("/devices/{id}/documents/{uuid}/cover", api.GetDocumentCover).Methods(http.MethodGet)
	apiRouter.HandleFunc("/devices/{id}/documents/{uuid}/link", api.LinkDocument).Methods(http.MethodPost)
	apiRouter.HandleFunc("/devices/{id}/documents/{uuid}/link", api.UnlinkDocument).Methods(http.MethodDelete)
	return httpx.HandlerWithCORS(router)
}
