package openlibrary

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ansel1/merry"
)

const (
	defaultBaseURL = "https://openlibrary.org"
	// Open Library asks clients to identify themselves; identified requests
	// get a higher rate limit than anonymous ones.
	userAgent = "reMarkableShelf (+https://github.com/kduong-dev/reMarkableShelf)"
	// MinimumQueryLength is the shortest query Open Library's search accepts.
	MinimumQueryLength = 3
)

type APIClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient builds an Open Library client. No API key is required.
func NewClient() *APIClient {
	return NewClientWithBaseURL(defaultBaseURL)
}

// NewClientWithBaseURL is NewClient with the Open Library base URL
// overridable, so tests can point it at an httptest server instead of the
// real API.
func NewClientWithBaseURL(baseURL string) *APIClient {
	return &APIClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: baseURL,
	}
}

func (api *APIClient) Search(query string) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrEmptyQuery
	}
	if utf8.RuneCountInString(query) < MinimumQueryLength {
		return nil, merry.Here(ErrQueryTooShort)
	}
	response, err := api.get(api.createSearchURL(query))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	// Open Library answers 422 for queries it refuses to run, such as a
	// lone stop word like "the"; that is the user's query, not an outage.
	if response.StatusCode == http.StatusUnprocessableEntity {
		return nil, merry.Here(ErrQueryRejected).WithMessagef("open library rejected query %q", query)
	}
	var parsed searchResponse
	if err := decodeJSON(response, &parsed); err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(parsed.Docs))
	for _, document := range parsed.Docs {
		results = append(results, document.toResult())
	}
	return results, nil
}

func (api *APIClient) EditionPageCount(isbn string) (int, error) {
	if isbn == "" {
		return 0, ErrEmptyISBN
	}
	// The /isbn endpoint redirects to the edition's canonical URL, which
	// the default client follows.
	response, err := api.get(api.baseURL + "/isbn/" + url.PathEscape(isbn) + ".json")
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return 0, merry.Here(ErrEditionNotFound).WithMessagef("no edition with isbn %s", isbn)
	}
	var parsed editionResponse
	if err := decodeJSON(response, &parsed); err != nil {
		return 0, err
	}
	return parsed.PageCount, nil
}

func (api *APIClient) createSearchURL(query string) string {
	values := make(url.Values)
	values.Set("q", query)
	values.Set("limit", "20")
	values.Set("fields", "key,title,author_name,isbn,cover_i,number_of_pages_median")
	return api.baseURL + "/search.json?" + values.Encode()
}

func (api *APIClient) get(requestURL string) (*http.Response, error) {
	request, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	request.Header.Set("User-Agent", userAgent)
	response, err := api.httpClient.Do(request)
	if err != nil {
		return nil, merry.Wrap(err).WithUserMessage("contacting Open Library")
	}
	return response, nil
}

func decodeJSON(response *http.Response, target any) error {
	if response.StatusCode != http.StatusOK {
		return merry.Here(ErrUpstreamUnavailable).WithMessagef("open library returned status %d", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return merry.Wrap(err).WithUserMessage("parsing Open Library response")
	}
	return nil
}
