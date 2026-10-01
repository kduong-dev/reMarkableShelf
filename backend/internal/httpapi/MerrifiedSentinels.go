package httpapi

import (
	"fmt"
	"net/http"

	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookfilestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/booksource"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicesync"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/documentstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/internetarchive"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/remarkable"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/sourcestore"
)

// merrifiedSentinels are the errors a client can cause, and how each is sent
// to them. devicesync.ErrDeviceAlreadyRegistered is left out because
// CreateDevice names the device already holding the host.
var merrifiedSentinels = httpx.MerrifiedSentinels{
	{Sentinel: bookstore.ErrBookNotFound, StatusCode: http.StatusNotFound, UserMessage: "no book with that id"},
	{Sentinel: bookstore.ErrInvalidPageCount, StatusCode: http.StatusBadRequest, UserMessage: "the page count must be at least 1"},
	{Sentinel: bookstore.ErrCurrentPageOutOfRange, StatusCode: http.StatusBadRequest, UserMessage: "the page can't be negative or past the end of the book"},
	{Sentinel: devicestore.ErrDeviceNotFound, StatusCode: http.StatusNotFound, UserMessage: "no device with that id"},
	{Sentinel: devicestore.ErrDeviceNameRequired, StatusCode: http.StatusBadRequest, UserMessage: "enter a name for the device"},
	{Sentinel: documentstore.ErrDocumentNotFound, StatusCode: http.StatusNotFound, UserMessage: "no document with that id on the device"},
	{Sentinel: documentstore.ErrCoverNotFound, StatusCode: http.StatusNotFound, UserMessage: "that document has no synced cover"},
	{Sentinel: devicesync.ErrDeviceDetailsRequired, StatusCode: http.StatusBadRequest, UserMessage: "enter the device's name and host"},
	{Sentinel: devicesync.ErrPasswordRequired, StatusCode: http.StatusBadRequest, UserMessage: "enter the tablet's password to pair it"},
	{Sentinel: remarkable.ErrWrongPassword, StatusCode: http.StatusBadRequest, UserMessage: "the tablet rejected that password"},
	{Sentinel: remarkable.ErrNotPaired, StatusCode: http.StatusBadGateway, UserMessage: "the tablet no longer accepts this server's key, so pair it again"},
	{Sentinel: remarkable.ErrKeyNotAccepted, StatusCode: http.StatusBadGateway, UserMessage: "the key was installed but the tablet still won't accept it"},
	{Sentinel: remarkable.ErrHostKeyChanged, StatusCode: http.StatusBadGateway, UserMessage: "the tablet identified itself differently than when it was paired; if you factory-reset or replaced it, pair it again, otherwise something on your network may be impersonating it"},
	{Sentinel: remarkable.ErrRestartFailed, StatusCode: http.StatusBadGateway, UserMessage: "books were copied to the tablet, but its reading app didn't restart to show them; press Sync now to try again"},
	{Sentinel: remarkable.ErrTabletUnreachable, StatusCode: http.StatusGatewayTimeout, UserMessage: "couldn't reach the tablet; it may be asleep or off the network"},
	{Sentinel: sourcestore.ErrSourceNotFound, StatusCode: http.StatusNotFound, UserMessage: "no ebook source with that id"},
	{Sentinel: sourcestore.ErrBuiltInSource, StatusCode: http.StatusBadRequest, UserMessage: "the Internet Archive source is built in; disable it instead"},
	{Sentinel: sourcestore.ErrSourceExists, StatusCode: http.StatusConflict, UserMessage: "that source is already added"},
	{Sentinel: sourcestore.ErrInvalidOrder, StatusCode: http.StatusBadRequest, UserMessage: "the order must list every source once"},
	{Sentinel: booksource.ErrUnknownKind, StatusCode: http.StatusBadRequest, UserMessage: "choose a plugin or an OPDS catalog"},
	{Sentinel: booksource.ErrInvalidURL, StatusCode: http.StatusBadRequest, UserMessage: "enter an http or https URL"},
	{Sentinel: booksource.ErrNotPlugin, StatusCode: http.StatusBadRequest, UserMessage: "no ebook source plugin answered at that URL"},
	{Sentinel: booksource.ErrNotOPDS, StatusCode: http.StatusBadRequest, UserMessage: "that URL isn't an OPDS catalog"},
	{Sentinel: booksource.ErrNotSearchable, StatusCode: http.StatusBadRequest, UserMessage: "that OPDS catalog can't be searched, so books can't be found in it"},
	{Sentinel: booksource.ErrNoEbook, StatusCode: http.StatusNotFound, UserMessage: "none of your ebook sources has this book"},
	{Sentinel: booksource.ErrEbookNotFound, StatusCode: http.StatusNotFound, UserMessage: "the source no longer has that ebook"},
	{Sentinel: booksource.ErrInvalidEbookID, StatusCode: http.StatusBadRequest, UserMessage: "that isn't an ebook from this source"},
	{Sentinel: booksource.ErrSourceUnavailable, StatusCode: http.StatusBadGateway, UserMessage: "the ebook source didn't answer, try again shortly"},
	{Sentinel: openlibrary.ErrEmptyQuery, StatusCode: http.StatusBadRequest, UserMessage: "enter a title, author or ISBN, or pick a genre"},
	{Sentinel: openlibrary.ErrQueryTooShort, StatusCode: http.StatusBadRequest, UserMessage: fmt.Sprintf("search for at least %d characters", openlibrary.MinimumQueryLength)},
	{Sentinel: openlibrary.ErrQueryRejected, StatusCode: http.StatusBadRequest, UserMessage: "Open Library can't search for that, try a more specific search"},
	{Sentinel: openlibrary.ErrInvalidSort, StatusCode: http.StatusBadRequest, UserMessage: "sort by relevance, rating, new or old"},
	{Sentinel: openlibrary.ErrInvalidLanguage, StatusCode: http.StatusBadRequest, UserMessage: "language must be a 3-letter code such as eng"},
	{Sentinel: openlibrary.ErrInvalidSubject, StatusCode: http.StatusBadRequest, UserMessage: "genre must be letters, numbers and spaces"},
	{Sentinel: openlibrary.ErrInvalidPage, StatusCode: http.StatusBadRequest, UserMessage: "page must be 1 or more"},
	{Sentinel: openlibrary.ErrInvalidYearRange, StatusCode: http.StatusBadRequest, UserMessage: "the published year range is invalid"},
	{Sentinel: openlibrary.ErrUpstreamUnavailable, StatusCode: http.StatusBadGateway, UserMessage: "Open Library is unavailable, try again shortly"},
	{Sentinel: openlibrary.ErrInvalidWorkID, StatusCode: http.StatusBadRequest, UserMessage: "that isn't an Open Library work id"},
	{Sentinel: openlibrary.ErrWorkNotFound, StatusCode: http.StatusNotFound, UserMessage: "Open Library has no work with that id"},
	{Sentinel: openlibrary.ErrNotPublicDomain, StatusCode: http.StatusNotFound, UserMessage: "only public domain books can be downloaded"},
	{Sentinel: internetarchive.ErrNoPublicEbook, StatusCode: http.StatusNotFound, UserMessage: "the Internet Archive has no free EPUB or PDF of that book"},
	{Sentinel: bookfilestore.ErrFileNotFound, StatusCode: http.StatusNotFound, UserMessage: "that book has no file saved on the server"},
	{Sentinel: bookfilestore.ErrUnsupportedFormat, StatusCode: http.StatusUnsupportedMediaType, UserMessage: "only EPUB and PDF files can be saved"},
	{Sentinel: bookfilestore.ErrStorageUnavailable, StatusCode: http.StatusBadGateway, UserMessage: "the file storage service is unavailable, try again shortly"},
	{Sentinel: internetarchive.ErrUpstreamUnavailable, StatusCode: http.StatusBadGateway, UserMessage: "the Internet Archive is unavailable, try again shortly"},
}
