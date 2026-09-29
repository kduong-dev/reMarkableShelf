package openlibrary

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/goutil/httpx"
)

const (
	defaultBaseURL = "https://openlibrary.org"
	// Open Library asks clients to identify themselves; identified requests
	// get a higher rate limit than anonymous ones.
	userAgent = "reMarkableShelf (+https://github.com/kduong-dev/reMarkableShelf)"
	// MinimumQueryLength is the shortest query Open Library's search accepts.
	MinimumQueryLength = 3
	// PageSize is how many results each page of a search holds.
	PageSize = 20
)

// workIDPattern matches a work's Open Library ID, such as OL893415W, so it
// can be searched for without adding clauses of its own.
var workIDPattern = regexp.MustCompile(`^OL[0-9]+W$`)

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

// searchSentinels maps the statuses Open Library answers a search with when
// the fault is the user's query, such as 422 for a lone stop word like "the".
var searchSentinels = map[int]error{
	http.StatusUnprocessableEntity: ErrQueryRejected,
}

var editionSentinels = map[int]error{
	http.StatusNotFound: ErrEditionNotFound,
}

func (api *APIClient) Search(input SearchInput) (SearchResults, error) {
	input.Query = strings.TrimSpace(input.Query)
	if err := input.validate(); err != nil {
		return SearchResults{}, err
	}
	var parsed searchResponse
	if err := api.getJSON(api.createSearchURL(input), searchSentinels, &parsed); err != nil {
		return SearchResults{}, err
	}
	results := make([]Result, 0, len(parsed.Docs))
	for _, document := range parsed.Docs {
		results = append(results, document.toResult())
	}
	return SearchResults{Results: results, Total: parsed.NumFound}, nil
}

func (api *APIClient) EditionPageCount(isbn string) (int, error) {
	if isbn == "" {
		return 0, ErrEmptyISBN
	}
	// The /isbn endpoint redirects to the edition's canonical URL, which
	// the default client follows.
	var parsed editionResponse
	if err := api.getJSON(api.baseURL+"/isbn/"+url.PathEscape(isbn)+".json", editionSentinels, &parsed); err != nil {
		return 0, err
	}
	return parsed.PageCount, nil
}

func (api *APIClient) PublicScans(openLibraryID string) (Scans, error) {
	if !workIDPattern.MatchString(openLibraryID) {
		return Scans{}, fmt.Errorf("%w %q", ErrInvalidWorkID, openLibraryID)
	}
	values := make(url.Values)
	values.Set("q", fmt.Sprintf("key:%q", "/works/"+openLibraryID))
	values.Set("fields", "key,title,ebook_access,ia")
	values.Set("limit", "1")
	var parsed searchResponse
	if err := api.getJSON(api.baseURL+"/search.json?"+values.Encode(), nil, &parsed); err != nil {
		return Scans{}, err
	}
	if len(parsed.Docs) == 0 {
		return Scans{}, fmt.Errorf("%w: %s", ErrWorkNotFound, openLibraryID)
	}
	document := parsed.Docs[0]
	if !document.isPublicDomain() || len(document.ArchiveIDs) == 0 {
		return Scans{}, fmt.Errorf("%w: %s", ErrNotPublicDomain, openLibraryID)
	}
	return Scans{Title: document.Title, ArchiveIDs: document.ArchiveIDs}, nil
}

func (api *APIClient) createSearchURL(input SearchInput) string {
	values := make(url.Values)
	values.Set("q", input.searchQuery())
	if input.Sort != SortRelevance {
		values.Set("sort", string(input.Sort))
	}
	values.Set("limit", fmt.Sprint(PageSize))
	if input.Page > 1 {
		values.Set("page", fmt.Sprint(input.Page))
	}
	values.Set("fields", "key,title,author_name,isbn,cover_i,number_of_pages_median,first_publish_year,ratings_average,ebook_access")
	return api.baseURL + "/search.json?" + values.Encode()
}

// getJSON fetches requestURL and decodes its JSON body into output. Any
// failure is reported as the sentinel matching the response's status in
// sentinelByStatusCode, or ErrUpstreamUnavailable, with the underlying error
// (for a bad status, the upstream status and body) kept as its cause.
func (api *APIClient) getJSON(requestURL string, sentinelByStatusCode map[int]error, output any) error {
	request, err := http.NewRequest(http.MethodGet, requestURL, nil)
	fatal.OnError(err)
	request.Header.Set("User-Agent", userAgent)
	response, err := api.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUpstreamUnavailable, err)
	}
	defer httpx.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusOK {
		sentinel, ok := sentinelByStatusCode[response.StatusCode]
		if !ok {
			sentinel = ErrUpstreamUnavailable
		}
		return fmt.Errorf("%w: %w", sentinel, httpx.ResponseError(response))
	}
	if err := json.NewDecoder(response.Body).Decode(output); err != nil {
		return fmt.Errorf("%w: %w", ErrUpstreamUnavailable, err)
	}
	return nil
}
