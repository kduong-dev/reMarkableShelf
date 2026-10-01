package booksource

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/internetarchive"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
)

// Factory opens the ebook sources the user has added, and checks new ones.
type Factory struct {
	openLibrary     openlibrary.API
	internetArchive internetarchive.API
	clients         clients
}

type NewFactoryInput struct {
	OpenLibrary     openlibrary.API
	InternetArchive internetarchive.API
}

func NewFactory(input NewFactoryInput) *Factory {
	return &Factory{
		openLibrary:     input.OpenLibrary,
		internetArchive: input.InternetArchive,
		clients: clients{
			lookup:   &http.Client{Timeout: 30 * time.Second},
			download: &http.Client{},
		},
	}
}

// Open is the Source an added ebook source reaches.
func (factory *Factory) Open(source models.EbookSource) (Source, error) {
	switch source.Kind {
	case models.EbookSourceKindInternetArchive:
		return internetArchive{openLibrary: factory.openLibrary, internetArchive: factory.internetArchive}, nil
	case models.EbookSourceKindPlugin:
		return plugin{baseURL: source.URL, clients: factory.clients}, nil
	case models.EbookSourceKindOPDS:
		return opds{searchURL: source.SearchURL, clients: factory.clients}, nil
	}
	return nil, fmt.Errorf("%w %q", ErrUnknownKind, source.Kind)
}

// Inspect checks that a plugin or OPDS catalog answers at rawURL, and
// returns it as an ebook source to add, named by itself.
func (factory *Factory) Inspect(ctx context.Context, kind models.EbookSourceKind, rawURL string) (models.EbookSource, error) {
	source := models.EbookSource{Kind: kind}
	var err error
	switch kind {
	case models.EbookSourceKindPlugin:
		source.URL, source.Name, err = inspectPlugin(ctx, factory.clients, rawURL)
	case models.EbookSourceKindOPDS:
		source.URL = rawURL
		source.Name, source.SearchURL, err = inspectOPDS(ctx, factory.clients, rawURL)
	default:
		err = fmt.Errorf("%w %q", ErrUnknownKind, kind)
	}
	return source, err
}

// Found is what one source found of a book, or why it couldn't look.
type Found struct {
	Source models.EbookSource
	Ebooks []Ebook
	Err    error
}

// FindAll asks every enabled source for ebooks of the book at once, and
// returns what each found, in the sources' order.
func (factory *Factory) FindAll(ctx context.Context, sources []models.EbookSource, query Query) []Found {
	var enabled []models.EbookSource
	for _, source := range sources {
		if source.Enabled {
			enabled = append(enabled, source)
		}
	}
	found := make([]Found, len(enabled))
	var group sync.WaitGroup
	for index, source := range enabled {
		group.Go(func() {
			found[index] = Found{Source: source}
			opened, err := factory.Open(source)
			if err == nil {
				found[index].Ebooks, err = opened.FindEbooks(ctx, query)
			}
			found[index].Err = err
		})
	}
	group.Wait()
	return found
}

// FindFirst asks the enabled sources in order for the book, and returns
// the first that has it with its preferred ebook. It returns ErrNoEbook
// when none has it, or ErrSourceUnavailable when every source failed.
func (factory *Factory) FindFirst(ctx context.Context, sources []models.EbookSource, query Query) (models.EbookSource, Ebook, error) {
	var failures []error
	asked := 0
	for _, source := range sources {
		if !source.Enabled {
			continue
		}
		asked++
		opened, err := factory.Open(source)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		ebooks, err := opened.FindEbooks(ctx, query)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", source.Name, err))
			continue
		}
		if len(ebooks) > 0 {
			return source, Preferred(ebooks), nil
		}
	}
	if asked > 0 && len(failures) == asked {
		return models.EbookSource{}, Ebook{}, errors.Join(append([]error{ErrSourceUnavailable}, failures...)...)
	}
	return models.EbookSource{}, Ebook{}, ErrNoEbook
}

// Preferred picks the ebook to fetch from a source's list: an EPUB, which
// reflows on the tablet, keeping its images if an edition does, since the
// tablet draws the first page, usually the cover, as its thumbnail; else
// the source's first file.
func Preferred(ebooks []Ebook) Ebook {
	best, bestRank := ebooks[0], -1
	for _, ebook := range ebooks {
		rank := 0
		if ebook.Format == models.FileTypeEPUB {
			rank = 1
			if !strings.Contains(strings.ToLower(ebook.Description), "no images") {
				rank = 2
			}
		}
		if rank > bestRank {
			best, bestRank = ebook, rank
		}
	}
	return best
}
