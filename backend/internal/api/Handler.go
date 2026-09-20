package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kqvd/reMarkableShelf/backend/internal/googlebooks"
	"github.com/kqvd/reMarkableShelf/backend/internal/remarkable"
	"github.com/kqvd/reMarkableShelf/backend/internal/store"
)

type Handler struct {
	Store         *store.Store
	GoogleBooks   googlebooks.API
	RemarkableCfg remarkable.Config
}

func NewHandler(handler *Handler) http.Handler {
	router := mux.NewRouter().StrictSlash(true)
	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/books", handler.ListBooks).Methods(http.MethodGet)
	apiRouter.HandleFunc("/books", handler.CreateBook).Methods(http.MethodPost)
	apiRouter.HandleFunc("/books/{id}", handler.GetBook).Methods(http.MethodGet)
	apiRouter.HandleFunc("/books/{id}", handler.UpdateBook).Methods(http.MethodPut)
	apiRouter.HandleFunc("/books/{id}", handler.DeleteBook).Methods(http.MethodDelete)
	apiRouter.HandleFunc("/search/books", handler.SearchBooks).Methods(http.MethodGet)
	apiRouter.HandleFunc("/devices", handler.ListDevices).Methods(http.MethodGet)
	apiRouter.HandleFunc("/devices", handler.CreateDevice).Methods(http.MethodPost)
	apiRouter.HandleFunc("/devices/{id}/sync", handler.SyncDevice).Methods(http.MethodPost)
	apiRouter.HandleFunc("/devices/{id}/documents", handler.ListDocuments).Methods(http.MethodGet)
	apiRouter.HandleFunc("/devices/{id}/documents/{uuid}/link", handler.LinkDocument).Methods(http.MethodPost)
	return HandlerWithCORS(router)
}
