package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookfilestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/booksource"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicesync"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/documentstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/internetarchive"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/sourcestore"
)

type NewHandlerInput struct {
	BookFileStore   bookfilestore.Store
	BookStore       bookstore.Store
	DeviceStore     devicestore.Store
	DocumentStore   documentstore.Store
	InternetArchive internetarchive.API
	OpenLibrary     openlibrary.API
	SourceStore     sourcestore.Store
	Syncer          *devicesync.Syncer
}

func NewHandler(input NewHandlerInput) http.Handler {
	api := &API{
		bookFileStore: input.BookFileStore,
		bookStore:     input.BookStore,
		deviceStore:   input.DeviceStore,
		documentStore: input.DocumentStore,
		ebookSources: booksource.NewFactory(booksource.NewFactoryInput{
			OpenLibrary:     input.OpenLibrary,
			InternetArchive: input.InternetArchive,
		}),
		openLibrary: input.OpenLibrary,
		sourceStore: input.SourceStore,
		syncer:      input.Syncer,
	}
	router := mux.NewRouter().StrictSlash(true)
	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/books", api.ListBooks).Methods(http.MethodGet)
	apiRouter.HandleFunc("/books", api.CreateBook).Methods(http.MethodPost)
	apiRouter.HandleFunc("/books/{id}", api.GetBook).Methods(http.MethodGet)
	apiRouter.HandleFunc("/books/{id}", api.UpdateBook).Methods(http.MethodPut)
	apiRouter.HandleFunc("/books/{id}", api.DeleteBook).Methods(http.MethodDelete)
	apiRouter.HandleFunc("/books/{id}/file", api.GetBookFile).Methods(http.MethodGet)
	apiRouter.HandleFunc("/books/{id}/file", api.UploadBookFile).Methods(http.MethodPut)
	apiRouter.HandleFunc("/books/{id}/file", api.DeleteBookFile).Methods(http.MethodDelete)
	apiRouter.HandleFunc("/books/{id}/file/fetch", api.FetchBookFile).Methods(http.MethodPost)
	apiRouter.HandleFunc("/books/{id}/ebooks", api.ListBookEbooks).Methods(http.MethodGet)
	apiRouter.HandleFunc("/books/{id}/file/content", api.DownloadBookFile).Methods(http.MethodGet)
	apiRouter.HandleFunc("/search/books", api.SearchBooks).Methods(http.MethodGet)
	apiRouter.HandleFunc("/search/books/{openLibraryId}/download", api.DownloadBook).Methods(http.MethodGet)
	apiRouter.HandleFunc("/sources", api.ListSources).Methods(http.MethodGet)
	apiRouter.HandleFunc("/sources", api.AddSource).Methods(http.MethodPost)
	apiRouter.HandleFunc("/sources/order", api.ReorderSources).Methods(http.MethodPut)
	apiRouter.HandleFunc("/sources/{id}", api.UpdateSource).Methods(http.MethodPut)
	apiRouter.HandleFunc("/sources/{id}", api.DeleteSource).Methods(http.MethodDelete)
	apiRouter.HandleFunc("/devices", api.ListDevices).Methods(http.MethodGet)
	apiRouter.HandleFunc("/devices", api.CreateDevice).Methods(http.MethodPost)
	apiRouter.HandleFunc("/devices/{id}", api.UpdateDevice).Methods(http.MethodPut)
	apiRouter.HandleFunc("/devices/{id}", api.DeleteDevice).Methods(http.MethodDelete)
	apiRouter.HandleFunc("/devices/{id}/pair", api.PairDevice).Methods(http.MethodPost)
	apiRouter.HandleFunc("/devices/{id}/sync", api.SyncDevice).Methods(http.MethodPost)
	apiRouter.HandleFunc("/devices/{id}/documents", api.ListDocuments).Methods(http.MethodGet)
	apiRouter.HandleFunc("/devices/{id}/folders", api.ListFolders).Methods(http.MethodGet)
	apiRouter.HandleFunc("/devices/{id}/documents/{uuid}/cover", api.GetDocumentCover).Methods(http.MethodGet)
	apiRouter.HandleFunc("/devices/{id}/documents/{uuid}/link", api.LinkDocument).Methods(http.MethodPost)
	apiRouter.HandleFunc("/devices/{id}/documents/{uuid}/link", api.UnlinkDocument).Methods(http.MethodDelete)
	apiRouter.HandleFunc("/devices/{id}/documents/{uuid}/not-a-book", api.MarkNotABook).Methods(http.MethodPost)
	apiRouter.HandleFunc("/devices/{id}/documents/{uuid}/not-a-book", api.UnmarkNotABook).Methods(http.MethodDelete)
	return httpx.HandlerWithCORS(router)
}
