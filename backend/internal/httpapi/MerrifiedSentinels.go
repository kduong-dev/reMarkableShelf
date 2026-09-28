package httpapi

import (
	"net/http"

	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/documentstore"
)

// merrifiedSentinels are the store errors a client can cause, and how each
// is sent to them.
var merrifiedSentinels = httpx.MerrifiedSentinels{
	{Sentinel: bookstore.ErrBookNotFound, StatusCode: http.StatusNotFound, UserMessage: "no book with that id"},
	{Sentinel: bookstore.ErrInvalidPageCount, StatusCode: http.StatusBadRequest, UserMessage: "the page count must be at least 1"},
	{Sentinel: bookstore.ErrCurrentPageOutOfRange, StatusCode: http.StatusBadRequest, UserMessage: "the page can't be negative or past the end of the book"},
	{Sentinel: devicestore.ErrDeviceNotFound, StatusCode: http.StatusNotFound, UserMessage: "no device with that id"},
	{Sentinel: devicestore.ErrDeviceNameRequired, StatusCode: http.StatusBadRequest, UserMessage: "enter a name for the device"},
	{Sentinel: documentstore.ErrDocumentNotFound, StatusCode: http.StatusNotFound, UserMessage: "no document with that id on the device"},
	{Sentinel: documentstore.ErrCoverNotFound, StatusCode: http.StatusNotFound, UserMessage: "that document has no synced cover"},
}

// sendErrorResponse sends err with the status code and user message of the
// store sentinel it wraps, including those returned through the syncer.
func sendErrorResponse(responseWriter http.ResponseWriter, err error) {
	httpx.SendErrorResponse(responseWriter, merrifiedSentinels.Merrify(err))
}
