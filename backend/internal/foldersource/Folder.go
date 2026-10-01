// Package foldersource is an ebook source plugin serving the EPUBs and PDFs
// in a folder, such as DRM-free purchases, which reMarkable Shelf can then
// fetch for the books they're of.
package foldersource

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/kduong-dev/reMarkableShelf/backend/pkg/sourceplugin"
)

var formats = map[string]sourceplugin.Format{
	".epub": sourceplugin.FormatEPUB,
	".pdf":  sourceplugin.FormatPDF,
}

// Folder serves the ebooks under its directory. An ebook's ID is its path
// from the directory, with forward slashes.
type Folder struct {
	name      string
	directory string
}

func New(name, directory string) *Folder {
	return &Folder{name: name, directory: directory}
}

func (folder *Folder) Manifest() sourceplugin.Manifest {
	return sourceplugin.Manifest{Name: folder.name, Description: "EPUBs and PDFs in " + folder.directory}
}

// FindEbooks lists the files whose names hold every word of the book's
// main title.
func (folder *Folder) FindEbooks(ctx context.Context, request sourceplugin.FindRequest) ([]sourceplugin.Ebook, error) {
	titleWords := words(mainTitle(request.Title))
	if len(titleWords) == 0 {
		return nil, nil
	}
	var ebooks []sourceplugin.Ebook
	err := filepath.WalkDir(folder.directory, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		format, ok := formats[strings.ToLower(filepath.Ext(entry.Name()))]
		if !ok || !containsAll(words(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))), titleWords) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(folder.directory, filePath)
		if err != nil {
			return err
		}
		ebooks = append(ebooks, sourceplugin.Ebook{
			ID:          filepath.ToSlash(relative),
			Title:       strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())),
			Format:      format,
			Size:        info.Size(),
			Description: filepath.ToSlash(relative),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", folder.directory, err)
	}
	return ebooks, nil
}

// OpenEbook opens an ebook by its path, refusing any path leading out of
// the directory or to a file that isn't an EPUB or PDF.
func (folder *Folder) OpenEbook(ctx context.Context, id string) (sourceplugin.Content, error) {
	cleaned := path.Clean("/" + id)[1:]
	format, ok := formats[strings.ToLower(path.Ext(cleaned))]
	if !ok || cleaned == "" || cleaned != id {
		return sourceplugin.Content{}, sourceplugin.ErrEbookNotFound
	}
	file, err := os.Open(filepath.Join(folder.directory, filepath.FromSlash(cleaned)))
	if errors.Is(err, fs.ErrNotExist) {
		return sourceplugin.Content{}, sourceplugin.ErrEbookNotFound
	}
	if err != nil {
		return sourceplugin.Content{}, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return sourceplugin.Content{}, err
	}
	return sourceplugin.Content{Body: file, Format: format, Size: info.Size()}, nil
}

func mainTitle(title string) string {
	if end := strings.IndexAny(title, ":;(["); end >= 0 {
		title = title[:end]
	}
	return title
}

func words(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(character rune) bool {
		return !unicode.IsLetter(character) && !unicode.IsNumber(character)
	})
}

func containsAll(haystack, needles []string) bool {
	present := make(map[string]bool, len(haystack))
	for _, word := range haystack {
		present[word] = true
	}
	for _, word := range needles {
		if !present[word] {
			return false
		}
	}
	return true
}
