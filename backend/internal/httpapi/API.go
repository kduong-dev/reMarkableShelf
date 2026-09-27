package httpapi

import (
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/remarkable"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/store"
)

type API struct {
	store            *store.Store
	openLibrary      openlibrary.API
	remarkableConfig remarkable.Config
}
