package httpapi

import (
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookfilestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/booksource"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicesync"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/documentstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/sourcestore"
)

type API struct {
	bookFileStore bookfilestore.Store
	bookStore     bookstore.Store
	deviceStore   devicestore.Store
	documentStore documentstore.Store
	ebookSources  *booksource.Factory
	openLibrary   openlibrary.API
	sourceStore   sourcestore.Store
	syncer        *devicesync.Syncer
}
