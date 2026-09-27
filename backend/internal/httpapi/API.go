package httpapi

import (
	"github.com/kqvd/reMarkableShelf/backend/internal/googlebooks"
	"github.com/kqvd/reMarkableShelf/backend/internal/remarkable"
	"github.com/kqvd/reMarkableShelf/backend/internal/store"
)

type API struct {
	store            *store.Store
	googleBooks      googlebooks.API
	remarkableConfig remarkable.Config
}
