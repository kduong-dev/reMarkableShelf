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
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	values := request.URL.Query()
	publishedFrom, err := optionalYear(values, "publishedFrom")
	if err != nil {
		return
	}
	publishedTo, err := optionalYear(values, "publishedTo")
	if err != nil {
		return
	}
	results, err := api.openLibrary.Search(openlibrary.SearchInput{
		Query:         values.Get("q"),
		Sort:          openlibrary.Sort(values.Get("sort")),
		Language:      values.Get("language"),
		PublishedFrom: publishedFrom,
		PublishedTo:   publishedTo,
	})
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, results)
}

// optionalYear parses the year in query parameter key, or 0 when it's absent.
func optionalYear(values url.Values, key string) (int, error) {
	raw := values.Get(key)
	if raw == "" {
		return 0, nil
	}
	year, err := strconv.Atoi(raw)
	if err != nil {
		return 0, merry.Here(ErrInvalidYear).WithUserMessagef("%s must be a year", key)
	}
	return year, nil
}
