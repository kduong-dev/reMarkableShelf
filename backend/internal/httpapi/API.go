package httpapi

import (
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicesync"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/store"
)

type API struct {
	store       *store.Store
	openLibrary openlibrary.API
	syncer      *devicesync.Syncer
}
