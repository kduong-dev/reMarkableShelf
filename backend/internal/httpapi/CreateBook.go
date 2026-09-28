package httpapi

import (
	"net/http"

	"github.com/ansel1/merry"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/goutil/logx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
)

func (api *API) CreateBook(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	body, err := httpx.DecodeJSONBody[models.Book](request)
	if err != nil {
		return
	}
	api.applyEditionPageCount(&body)
	book, err := api.store.Books.Create(body)
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusCreated, book)
}

// applyEditionPageCount replaces the page count sent by the client, usually
// the median across a work's editions, with the exact count of the edition
// matching the book's ISBN. A failed lookup keeps the client's count rather
// than failing the create, since the page count is only a convenience.
func (api *API) applyEditionPageCount(book *models.Book) {
	if book.ISBN == "" {
		return
	}
	pageCount, err := api.openLibrary.EditionPageCount(book.ISBN)
	if err != nil {
		if !merry.Is(err, openlibrary.ErrEditionNotFound) {
			logx.Warnf("looking up page count for isbn %s: %v", book.ISBN, err)
		}
		return
	}
	if pageCount > 0 {
		book.PageCount = &pageCount
	}
}
