package booksource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/goutil/httpx"
)

const userAgent = "reMarkableShelf (+https://github.com/kduong-dev/reMarkableShelf)"

// clients are the HTTP clients plugins and catalogs are reached with:
// lookups are capped, while downloads, which can run long over a slow
// link, end with their request's context.
type clients struct {
	lookup   *http.Client
	download *http.Client
}

// checkURL accepts only web URLs, so a source can't point the server at a
// file or another scheme.
func checkURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("%w: %q", ErrInvalidURL, raw)
	}
	return parsed, nil
}

func newRequest(ctx context.Context, method, requestURL string, body io.Reader) *http.Request {
	request, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	fatal.OnError(err)
	request.Header.Set("User-Agent", userAgent)
	return request
}

// getBody fetches the request and returns its response when it answers
// 200, reporting other statuses with httpx.ResponseError. A GET answered
// with a gateway error is tried once more after retryDelay, since public
// catalogs such as Project Gutenberg's often time out once and then answer.
func getBody(client *http.Client, request *http.Request) (*http.Response, error) {
	for attempt := 1; ; attempt++ {
		response, err := client.Do(request)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrSourceUnavailable, err)
		}
		if response.StatusCode == http.StatusOK {
			return response, nil
		}
		if attempt == 1 && request.Method == http.MethodGet && gatewayErrors[response.StatusCode] {
			httpx.DrainAndClose(response.Body)
			select {
			case <-time.After(retryDelay):
				continue
			case <-request.Context().Done():
				return nil, fmt.Errorf("%w: %w", ErrSourceUnavailable, request.Context().Err())
			}
		}
		defer httpx.DrainAndClose(response.Body)
		return nil, fmt.Errorf("%w: %w", ErrSourceUnavailable, httpx.ResponseError(response))
	}
}

// gatewayErrors are the statuses a lookup is retried after.
var gatewayErrors = map[int]bool{
	http.StatusBadGateway:         true,
	http.StatusServiceUnavailable: true,
	http.StatusGatewayTimeout:     true,
}

// retryDelay is how long a lookup waits before trying again; tests shorten
// it.
var retryDelay = time.Second

// getJSON decodes the 200 response to request into output.
func getJSON(client *http.Client, request *http.Request, output any) error {
	response, err := getBody(client, request)
	if err != nil {
		return err
	}
	defer httpx.DrainAndClose(response.Body)
	if err := json.NewDecoder(response.Body).Decode(output); err != nil {
		return fmt.Errorf("%w: %w", ErrSourceUnavailable, err)
	}
	return nil
}

// openDownload starts fetching requestURL, reporting a 404 as
// ErrEbookNotFound.
func openDownload(client *http.Client, request *http.Request) (Download, error) {
	response, err := client.Do(request)
	if err != nil {
		return Download{}, fmt.Errorf("%w: %w", ErrSourceUnavailable, err)
	}
	if response.StatusCode == http.StatusNotFound {
		defer httpx.DrainAndClose(response.Body)
		return Download{}, ErrEbookNotFound
	}
	if response.StatusCode != http.StatusOK {
		defer httpx.DrainAndClose(response.Body)
		return Download{}, fmt.Errorf("%w: %w", ErrSourceUnavailable, httpx.ResponseError(response))
	}
	return Download{Body: response.Body, ContentLength: response.ContentLength}, nil
}
