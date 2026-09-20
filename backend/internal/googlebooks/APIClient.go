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
	apiKey     string
}

// NewClient builds a Google Books client. apiKey may be empty, in which
// case requests are unauthenticated and share Google's low, per-IP-address
// anonymous quota — fine for local/occasional use, but requests can fail
// with a 429 if that shared quota is already exhausted. Passing a free
// Google Books API key (https://console.cloud.google.com/apis/credentials)
// gets its own, much higher daily quota instead.
func NewClient(apiKey string) *APIClient {
	return NewClientWithBaseURL(defaultSearchURL, apiKey)
}

// NewClientWithBaseURL is NewClient with the Google Books API's base URL
// overridable, so tests can point it at an httptest server instead of the
// real API.
func NewClientWithBaseURL(baseURL, apiKey string) *APIClient {
	return &APIClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: baseURL,
		apiKey:  apiKey,
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
	if api.apiKey != "" {
		values.Set("key", api.apiKey)
	}
	return api.baseURL + "?" + values.Encode()
}
