package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/kqvd/reMarkableShelf/backend/internal/httpx"
	"github.com/kqvd/reMarkableShelf/backend/internal/models"
	"github.com/kqvd/reMarkableShelf/backend/internal/remarkable"
)

// SyncDevice SSHes into the device, upserts every document it finds, and
// best-effort auto-links pdf/epub documents to an existing book by exact
// (case-insensitive) title match. Notebooks are stored but never
// auto-linked — see models.FileType.
func (handler *Handler) SyncDevice(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	vars := mux.Vars(request)
	deviceID := vars["id"]
	device, err := handler.Store.GetDevice(deviceID)
	if err != nil {
		return
	}
	documents, err := remarkable.ListDocuments(device.Host, handler.RemarkableCfg)
	if err != nil {
		return
	}
	err = handler.Store.UpsertDocuments(device.ID, documents)
	if err != nil {
		return
	}
	err = handler.Store.TouchDeviceSyncedAt(device.ID, time.Now().UTC())
	if err != nil {
		return
	}
	err = handler.autoLinkDocuments(device.ID)
	if err != nil {
		return
	}
	syncedDocuments, err := handler.Store.ListDocumentsByDevice(device.ID)
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, syncedDocuments)
}

func (handler *Handler) autoLinkDocuments(deviceID string) error {
	documents, err := handler.Store.ListDocumentsByDevice(deviceID)
	if err != nil {
		return err
	}
	books, err := handler.Store.ListBooks()
	if err != nil {
		return err
	}
	booksByTitle := make(map[string]models.Book, len(books))
	for _, book := range books {
		booksByTitle[strings.ToLower(strings.TrimSpace(book.Title))] = book
	}
	for _, document := range documents {
		if document.LinkedBookID != nil || document.FileType == models.FileTypeNotebook {
			continue
		}
		book, found := booksByTitle[strings.ToLower(strings.TrimSpace(document.Title))]
		if !found {
			continue
		}
		if err := handler.Store.LinkDocumentToBook(deviceID, document.UUID, book.ID); err != nil {
			return err
		}
	}
	return nil
}
