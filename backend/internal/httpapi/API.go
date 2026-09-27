package httpapi

import (
	"github.com/kqvd/reMarkableShelf/backend/internal/openlibrary"
	"github.com/kqvd/reMarkableShelf/backend/internal/remarkable"
	"github.com/kqvd/reMarkableShelf/backend/internal/store"
)

type API struct {
	store            *store.Store
	openLibrary      openlibrary.API
	remarkableConfig remarkable.Config
}
