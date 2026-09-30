package devicesync

import (
	"regexp"
	"strings"

	"github.com/kduong-dev/goutil/logx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

// importDocuments adds each PDF or EPUB on the device that no book is
// linked to as a book of its own, as is, and links it, so everything read
// on the tablet shows in the library; the user can match it to Open
// Library later. It leaves out documents in the trash, ones marked not a
// book, and ones the user unlinked or whose book they deleted.
func (syncer *Syncer) importDocuments(deviceID string, folders []models.RemarkableFolder) error {
	documents, err := syncer.documentStore.ListByDevice(deviceID)
	if err != nil {
		return err
	}
	parents := make(map[string]string, len(folders))
	for _, folder := range folders {
		parents[folder.UUID] = folder.ParentUUID
	}
	imported := 0
	for _, document := range documents {
		if document.FileType == models.FileTypeNotebook || document.LinkedBookID != nil ||
			document.AutoLinkDismissed || document.NotABook ||
			document.ParentUUID == models.TrashFolderUUID || inTrash(parents, document.ParentUUID) {
			continue
		}
		book, err := syncer.bookStore.Create(importedBook(document))
		if err != nil {
			return err
		}
		if err := syncer.documentStore.LinkToBook(deviceID, document.UUID, book.ID); err != nil {
			return err
		}
		imported++
	}
	if imported > 0 {
		logx.Noticef("imported %d books from device %s", imported, deviceID)
	}
	return nil
}

// importedBook is the book a tablet document is added as. An EPUB names
// its own title and author reliably; a PDF's embedded title is often a
// leftover such as "Microsoft Word - doc1", so a PDF takes its file name.
func importedBook(document models.RemarkableDocument) models.Book {
	book := models.Book{Title: titleFromFileName(document.Title), Source: models.SourceRemarkable}
	if document.FileType == models.FileTypeEPUB && document.BookTitle != "" {
		book.Title = document.BookTitle
		book.Author = document.BookAuthor
	}
	return book
}

var (
	fileExtension   = regexp.MustCompile(`(?i)\.(pdf|epub)$`)
	bracketedNote   = regexp.MustCompile(`\s*[(\[{][^)\]}]*[)\]}]`)
	underscoreColon = regexp.MustCompile(`_\s`)
	repeatedSpace   = regexp.MustCompile(`\s+`)
)

// titleFromFileName tidies a file name into a title: without its
// extension or bracketed notes such as "(2014)" or "(z-lib.org)", and with
// the underscores file names get in place of colons, as in
// "Crucial Conversations_ Tools", made colons again.
func titleFromFileName(name string) string {
	title := fileExtension.ReplaceAllString(name, "")
	title = bracketedNote.ReplaceAllString(title, "")
	title = underscoreColon.ReplaceAllString(title, ": ")
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.Trim(repeatedSpace.ReplaceAllString(title, " "), " -")
	if title == "" {
		return name
	}
	return title
}
