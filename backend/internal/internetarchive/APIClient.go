package internetarchive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/goutil/httpx"
)

const (
	defaultBaseURL = "https://archive.org"
	userAgent      = "reMarkableShelf (+https://github.com/kduong-dev/reMarkableShelf)"
	// maximumIdentifiers caps how many of a work's scans are searched, as
	// popular works have hundreds.
	maximumIdentifiers = 120
	// identifiersPerSearch keeps each search under the archive's limit on
	// the length of a query.
	identifiersPerSearch = 40
	// maximumItemsChecked caps how many public scans have their files listed
	// while looking for an EPUB.
	maximumItemsChecked = 5
	// gutenbergCollection holds Project Gutenberg's transcriptions, whose
	// EPUBs are typeset text rather than OCR of a scan.
	gutenbergCollection = "gutenberg"
)

// identifierPattern matches an Internet Archive identifier, so it can be
// searched for without adding clauses of its own.
var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// fileFormats are the ebook formats by the name the archive lists files under.
var fileFormats = map[string]Format{
	"EPUB":     FormatEPUB,
	"Text PDF": FormatPDF,
	"PDF":      FormatPDF,
}

type APIClient struct {
	httpClient     *http.Client
	downloadClient *http.Client
	baseURL        string
}

// NewClient builds an Internet Archive client. No API key is required.
func NewClient() *APIClient {
	return NewClientWithBaseURL(defaultBaseURL)
}

// NewClientWithBaseURL is NewClient with the Internet Archive base URL
// overridable, so tests can point it at an httptest server.
func NewClientWithBaseURL(baseURL string) *APIClient {
	return &APIClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		// Scans run to tens of megabytes, so downloads get longer, and are
		// cut short sooner when the caller's context ends.
		downloadClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
		baseURL: baseURL,
	}
}

func (api *APIClient) FindEbook(identifiers []string) (Ebook, error) {
	items, err := api.publicItems(identifiers)
	if err != nil {
		return Ebook{}, err
	}
	var fallback *Ebook
	for _, item := range items[:min(len(items), maximumItemsChecked)] {
		ebooks, err := api.ebooks(item.Identifier)
		if err != nil {
			return Ebook{}, err
		}
		for _, ebook := range ebooks {
			if ebook.Format == FormatEPUB {
				return ebook, nil
			}
			if fallback == nil {
				fallback = &ebook
			}
		}
	}
	if fallback == nil {
		return Ebook{}, ErrNoPublicEbook
	}
	return *fallback, nil
}

func (api *APIClient) OpenEbook(ctx context.Context, ebook Ebook) (Download, error) {
	requestURL := api.baseURL + "/download/" + url.PathEscape(ebook.Identifier) + "/" + escapePath(ebook.FileName)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	fatal.OnError(err)
	request.Header.Set("User-Agent", userAgent)
	response, err := api.downloadClient.Do(request)
	if err != nil {
		return Download{}, fmt.Errorf("%w: %w", ErrUpstreamUnavailable, err)
	}
	if response.StatusCode != http.StatusOK {
		defer httpx.DrainAndClose(response.Body)
		return Download{}, fmt.Errorf("%w: %w", ErrUpstreamUnavailable, httpx.ResponseError(response))
	}
	return Download{Body: response.Body, ContentLength: response.ContentLength}, nil
}

// publicItems searches for the scans among identifiers that anyone can
// download, putting Project Gutenberg's first.
func (api *APIClient) publicItems(identifiers []string) ([]searchDocument, error) {
	var valid []string
	for _, identifier := range identifiers {
		if identifierPattern.MatchString(identifier) {
			valid = append(valid, identifier)
		}
	}
	if len(valid) == 0 {
		return nil, ErrNoPublicEbook
	}
	var items []searchDocument
	for batch := range slices.Chunk(valid[:min(len(valid), maximumIdentifiers)], identifiersPerSearch) {
		found, err := api.searchPublicItems(batch)
		if err != nil {
			return nil, err
		}
		items = append(items, found...)
	}
	slices.SortStableFunc(items, func(first, second searchDocument) int {
		return second.priority() - first.priority()
	})
	return items, nil
}

func (api *APIClient) searchPublicItems(identifiers []string) ([]searchDocument, error) {
	values := make(url.Values)
	values.Set("q", fmt.Sprintf("identifier:(%s) AND mediatype:texts AND NOT access-restricted-item:true", strings.Join(identifiers, " OR ")))
	values.Add("fl[]", "identifier")
	values.Add("fl[]", "collection")
	values.Set("rows", fmt.Sprint(len(identifiers)))
	values.Set("output", "json")
	var parsed searchResponse
	if err := api.getJSON(api.baseURL+"/advancedsearch.php?"+values.Encode(), &parsed); err != nil {
		return nil, err
	}
	// The archive answers a query it can't run with a 200 carrying an error.
	if parsed.Error != "" {
		return nil, fmt.Errorf("%w: %s", ErrUpstreamUnavailable, parsed.Error)
	}
	return parsed.Response.Docs, nil
}

// ebooks lists the item's public EPUB and PDF files.
func (api *APIClient) ebooks(identifier string) ([]Ebook, error) {
	var parsed filesResponse
	if err := api.getJSON(api.baseURL+"/metadata/"+url.PathEscape(identifier)+"/files", &parsed); err != nil {
		return nil, err
	}
	var ebooks []Ebook
	for _, file := range parsed.Result {
		format, ok := fileFormats[file.Format]
		if !ok || file.Private == "true" {
			continue
		}
		ebooks = append(ebooks, Ebook{Identifier: identifier, FileName: file.Name, Format: format})
	}
	return ebooks, nil
}

func (api *APIClient) getJSON(requestURL string, output any) error {
	request, err := http.NewRequest(http.MethodGet, requestURL, nil)
	fatal.OnError(err)
	request.Header.Set("User-Agent", userAgent)
	response, err := api.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUpstreamUnavailable, err)
	}
	defer httpx.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %w", ErrUpstreamUnavailable, httpx.ResponseError(response))
	}
	if err := json.NewDecoder(response.Body).Decode(output); err != nil {
		return fmt.Errorf("%w: %w", ErrUpstreamUnavailable, err)
	}
	return nil
}

// escapePath escapes each segment of a file name, which can sit in a folder
// of its item.
func escapePath(fileName string) string {
	segments := strings.Split(fileName, "/")
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}
