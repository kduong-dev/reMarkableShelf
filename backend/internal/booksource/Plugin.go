package booksource

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/pkg/sourceplugin"
)

// plugin is a web service answering the sourceplugin contract at baseURL.
type plugin struct {
	baseURL string
	clients clients
}

func (source plugin) FindEbooks(ctx context.Context, query Query) ([]Ebook, error) {
	body := fatal.UnlessMarshal(sourceplugin.FindRequest{
		Title:         query.Title,
		Author:        query.Author,
		ISBN:          query.ISBN,
		OpenLibraryID: query.OpenLibraryID,
	})
	request := newRequest(ctx, http.MethodPost, source.baseURL+"/ebooks/find", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	var found sourceplugin.FindResponse
	if err := getJSON(source.clients.lookup, request, &found); err != nil {
		return nil, err
	}
	var ebooks []Ebook
	for _, ebook := range found.Ebooks {
		format := models.FileType(ebook.Format)
		if ebook.ID == "" || (format != models.FileTypeEPUB && format != models.FileTypePDF) {
			continue
		}
		ebooks = append(ebooks, Ebook{
			ID:          ebook.ID,
			Title:       ebook.Title,
			Author:      ebook.Author,
			Format:      format,
			Size:        ebook.Size,
			Description: ebook.Description,
		})
	}
	return ebooks, nil
}

func (source plugin) OpenEbook(ctx context.Context, ebookID string) (Download, error) {
	if ebookID == "" {
		return Download{}, ErrInvalidEbookID
	}
	request := newRequest(ctx, http.MethodGet, source.baseURL+"/ebooks/"+url.PathEscape(ebookID)+"/content", nil)
	return openDownload(source.clients.download, request)
}

// inspectPlugin reads the manifest of the plugin at rawURL, returning its
// base URL and name.
func inspectPlugin(ctx context.Context, clients clients, rawURL string) (string, string, error) {
	parsed, err := checkURL(rawURL)
	if err != nil {
		return "", "", err
	}
	baseURL := strings.TrimSuffix(parsed.String(), "/")
	var manifest sourceplugin.Manifest
	if err := getJSON(clients.lookup, newRequest(ctx, http.MethodGet, baseURL+"/manifest", nil), &manifest); err != nil {
		return "", "", fmt.Errorf("%w: %w", ErrNotPlugin, err)
	}
	if strings.TrimSpace(manifest.Name) == "" {
		return "", "", fmt.Errorf("%w: its manifest has no name", ErrNotPlugin)
	}
	return baseURL, strings.TrimSpace(manifest.Name), nil
}
