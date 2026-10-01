package booksource

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

const (
	acquisitionRel = "http://opds-spec.org/acquisition"
	// maximumEntriesFollowed caps how many search results are opened to
	// find their files, when a catalog lists them on each book's own page.
	maximumEntriesFollowed = 3
	maximumEbooks          = 10
)

// opdsTypes are the acquisition link types kept, by their file type.
var opdsTypes = map[string]models.FileType{
	"application/epub+zip": models.FileTypeEPUB,
	"application/pdf":      models.FileTypePDF,
}

// optionalTemplateParameter matches OpenSearch parameters other than the
// search terms, such as {startPage?}, which are left empty.
var optionalTemplateParameter = regexp.MustCompile(`\{[^}]*\}`)

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Title   string      `xml:"title"`
	Links   []atomLink  `xml:"link"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title   string `xml:"title"`
	Authors []struct {
		Name string `xml:"name"`
	} `xml:"author"`
	Links []atomLink `xml:"link"`
}

type atomLink struct {
	Rel    string `xml:"rel,attr"`
	Type   string `xml:"type,attr"`
	Href   string `xml:"href,attr"`
	Title  string `xml:"title,attr"`
	Length int64  `xml:"length,attr"`
}

type openSearchDescription struct {
	URLs []struct {
		Type     string `xml:"type,attr"`
		Template string `xml:"template,attr"`
	} `xml:"Url"`
}

// opds searches an OPDS 1 catalog with its OpenSearch template, whose
// ebook IDs are the files' URLs.
type opds struct {
	searchURL string
	clients   clients
}

func (source opds) FindEbooks(ctx context.Context, query Query) ([]Ebook, error) {
	terms := strings.TrimSpace(mainTitle(query.Title) + " " + query.Author)
	if terms == "" {
		return nil, nil
	}
	searchURL := strings.ReplaceAll(source.searchURL, "{searchTerms}", url.QueryEscape(terms))
	searchURL = optionalTemplateParameter.ReplaceAllString(searchURL, "")
	results, err := source.feed(ctx, searchURL)
	if err != nil {
		return nil, err
	}
	var ebooks []Ebook
	followed := 0
	for _, entry := range results.Entries {
		if !sameTitle(query.Title, entry.Title) {
			continue
		}
		found := acquisitions(searchURL, entry)
		if len(found) == 0 && followed < maximumEntriesFollowed {
			if detailsURL := catalogLink(searchURL, entry); detailsURL != "" {
				followed++
				details, err := source.feed(ctx, detailsURL)
				if err != nil {
					return nil, err
				}
				for _, edition := range details.Entries {
					if sameTitle(query.Title, edition.Title) {
						found = append(found, acquisitions(detailsURL, edition)...)
					}
				}
			}
		}
		ebooks = appendNew(ebooks, found)
		if len(ebooks) >= maximumEbooks {
			return ebooks[:maximumEbooks], nil
		}
	}
	return ebooks, nil
}

func (source opds) OpenEbook(ctx context.Context, ebookID string) (Download, error) {
	if _, err := checkURL(ebookID); err != nil {
		return Download{}, fmt.Errorf("%w: %w", ErrInvalidEbookID, err)
	}
	return openDownload(source.clients.download, newRequest(ctx, http.MethodGet, ebookID, nil))
}

func (source opds) feed(ctx context.Context, feedURL string) (atomFeed, error) {
	return readFeed(ctx, source.clients, feedURL)
}

func readFeed(ctx context.Context, clients clients, feedURL string) (atomFeed, error) {
	response, err := getBody(clients.lookup, newRequest(ctx, http.MethodGet, feedURL, nil))
	if err != nil {
		return atomFeed{}, err
	}
	defer httpx.DrainAndClose(response.Body)
	var feed atomFeed
	if err := xml.NewDecoder(response.Body).Decode(&feed); err != nil {
		return atomFeed{}, fmt.Errorf("%w: reading %s: %w", ErrSourceUnavailable, feedURL, err)
	}
	return feed, nil
}

// inspectOPDS reads the catalog at rawURL, returning its name and search
// template.
func inspectOPDS(ctx context.Context, clients clients, rawURL string) (string, string, error) {
	parsed, err := checkURL(rawURL)
	if err != nil {
		return "", "", err
	}
	root, err := readFeed(ctx, clients, parsed.String())
	if err != nil {
		return "", "", fmt.Errorf("%w: %w", ErrNotOPDS, err)
	}
	name := strings.TrimSpace(root.Title)
	if name == "" {
		name = parsed.Host
	}
	for _, link := range root.Links {
		if link.Rel != "search" {
			continue
		}
		href := resolve(parsed.String(), link.Href)
		if strings.Contains(href, "{searchTerms}") {
			return name, href, nil
		}
		if strings.Contains(link.Type, "opensearchdescription") {
			if template, err := searchTemplate(ctx, clients, href); err == nil {
				return name, template, nil
			}
		}
	}
	return "", "", ErrNotSearchable
}

// searchTemplate reads an OpenSearch description for its OPDS search URL.
func searchTemplate(ctx context.Context, clients clients, descriptionURL string) (string, error) {
	response, err := getBody(clients.lookup, newRequest(ctx, http.MethodGet, descriptionURL, nil))
	if err != nil {
		return "", err
	}
	defer httpx.DrainAndClose(response.Body)
	var description openSearchDescription
	if err := xml.NewDecoder(response.Body).Decode(&description); err != nil {
		return "", fmt.Errorf("%w: %w", ErrNotSearchable, err)
	}
	for _, candidate := range description.URLs {
		if strings.Contains(candidate.Type, "atom+xml") && strings.Contains(candidate.Template, "{searchTerms}") {
			return resolve(descriptionURL, candidate.Template), nil
		}
	}
	return "", ErrNotSearchable
}

// acquisitions lists an entry's EPUB and PDF files.
func acquisitions(feedURL string, entry atomEntry) []Ebook {
	author := ""
	if len(entry.Authors) > 0 {
		author = firstNameFirst(entry.Authors[0].Name)
	}
	var ebooks []Ebook
	for _, link := range entry.Links {
		format, ok := opdsTypes[strings.TrimSpace(strings.Split(link.Type, ";")[0])]
		if !ok || !strings.HasPrefix(link.Rel, acquisitionRel) {
			continue
		}
		ebooks = append(ebooks, Ebook{
			ID:          resolve(feedURL, link.Href),
			Title:       strings.TrimSpace(entry.Title),
			Author:      author,
			Format:      format,
			Size:        link.Length,
			Description: link.Title,
		})
	}
	return ebooks
}

// catalogLink is the feed describing an entry in full, which some
// catalogs link to from search results instead of listing its files.
func catalogLink(feedURL string, entry atomEntry) string {
	for _, link := range entry.Links {
		if strings.Contains(link.Type, "atom+xml") && (link.Rel == "subsection" || link.Rel == "alternate" || link.Rel == "") {
			return resolve(feedURL, link.Href)
		}
	}
	return ""
}

func appendNew(ebooks, found []Ebook) []Ebook {
	for _, ebook := range found {
		duplicate := false
		for _, existing := range ebooks {
			duplicate = duplicate || existing.ID == ebook.ID
		}
		if !duplicate {
			ebooks = append(ebooks, ebook)
		}
	}
	return ebooks
}

func resolve(baseURL, href string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return href
	}
	reference, err := url.Parse(href)
	if err != nil {
		return href
	}
	return base.ResolveReference(reference).String()
}
