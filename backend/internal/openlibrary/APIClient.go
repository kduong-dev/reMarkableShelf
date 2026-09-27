package openlibrary

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/ansel1/merry"
)

const (
	defaultSearchURL = "https://openlibrary.org/search.json"
	// Open Library asks clients to identify themselves; identified requests
	// get a higher rate limit than anonymous ones.
	userAgent = "reMarkableShelf (+https://github.com/kduong-dev/reMarkableShelf)"
)

type APIClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient builds an Open Library client. No API key is required.
func NewClient() *APIClient {
	return NewClientWithBaseURL(defaultSearchURL)
}

// NewClientWithBaseURL is NewClient with the Open Library search URL
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
	if query == "" {
		return nil, ErrEmptyQuery
	}
	request, err := http.NewRequest(http.MethodGet, api.createRequestURL(query), nil)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	request.Header.Set("User-Agent", userAgent)
	response, err := api.httpClient.Do(request)
	if err != nil {
		return nil, merry.Wrap(err).WithUserMessage("contacting Open Library")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, merry.Here(ErrUpstreamUnavailable).WithMessagef("open library returned status %d", response.StatusCode)
	}
	var parsed searchResponse
	if err := json.NewDecoder(response.Body).Decode(&parsed); err != nil {
		return nil, merry.Wrap(err).WithUserMessage("parsing Open Library response")
	}
	results := make([]Result, 0, len(parsed.Docs))
	for _, document := range parsed.Docs {
		results = append(results, document.toResult())
	}
	return results, nil
}

func (api *APIClient) createRequestURL(query string) string {
	values := make(url.Values)
	values.Set("q", query)
	values.Set("limit", "20")
	values.Set("fields", "key,title,author_name,isbn,cover_i")
	return api.baseURL + "?" + values.Encode()
}
