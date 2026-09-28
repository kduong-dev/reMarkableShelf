package httpapi

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/ansel1/merry"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
)

func (api *API) SearchBooks(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	values := request.URL.Query()
	publishedFrom, err := optionalNumber(values, "publishedFrom")
	if err != nil {
		return
	}
	publishedTo, err := optionalNumber(values, "publishedTo")
	if err != nil {
		return
	}
	page, err := optionalNumber(values, "page")
	if err != nil {
		return
	}
	results, err := api.openLibrary.Search(openlibrary.SearchInput{
		Query:         values.Get("q"),
		Subject:       values.Get("subject"),
		Sort:          openlibrary.Sort(values.Get("sort")),
		Language:      values.Get("language"),
		PublishedFrom: publishedFrom,
		PublishedTo:   publishedTo,
		Page:          page,
	})
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, results)
}

// optionalNumber parses the whole number in query parameter key, or 0 when
// it's absent.
func optionalNumber(values url.Values, key string) (int, error) {
	raw := values.Get(key)
	if raw == "" {
		return 0, nil
	}
	year, err := strconv.Atoi(raw)
	if err != nil {
		return 0, merry.Here(ErrInvalidNumber).WithUserMessagef("%s must be a whole number", key)
	}
	return year, nil
}
