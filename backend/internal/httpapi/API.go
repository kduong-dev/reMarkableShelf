package httpapi

import (
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicesync"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/documentstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/internetarchive"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
)

type API struct {
	bookStore       bookstore.Store
	deviceStore     devicestore.Store
	documentStore   documentstore.Store
	internetArchive internetarchive.API
	openLibrary     openlibrary.API
	syncer          *devicesync.Syncer
}
