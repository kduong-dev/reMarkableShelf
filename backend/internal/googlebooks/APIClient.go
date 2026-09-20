package googlebooks

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/ansel1/merry"
)

const defaultSearchURL = "https://www.googleapis.com/books/v1/volumes"

type APIClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient() *APIClient {
	return NewClientWithBaseURL(defaultSearchURL)
}

// NewClientWithBaseURL is NewClient with the Google Books API's base URL
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
	requestURL := api.createRequestURL(query)
	response, err := api.httpClient.Get(requestURL)
	if err != nil {
		return nil, merry.Wrap(err).WithUserMessage("contacting Google Books")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, merry.Here(ErrUpstreamUnavailable).WithMessagef("google books returned status %d", response.StatusCode)
	}
	var parsed searchResponse
	if err := json.NewDecoder(response.Body).Decode(&parsed); err != nil {
		return nil, merry.Wrap(err).WithUserMessage("parsing Google Books response")
	}
	results := make([]Result, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		results = append(results, item.toResult())
	}
	return results, nil
}

func (api *APIClient) createRequestURL(query string) string {
	values := make(url.Values)
	values.Set("q", query)
	values.Set("maxResults", "20")
	return api.baseURL + "?" + values.Encode()
}
