package httpapi

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/logx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/internetarchive"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
)

var contentTypes = map[string]string{
	"epub": "application/epub+zip",
	"pdf":  "application/pdf",
}

// DownloadBook streams a public domain work's ebook from the Internet
// Archive, named after the work's title.
func (api *API) DownloadBook(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	scans, ebook, download, err := api.openPublicEbook(request.Context(), mux.Vars(request)["openLibraryId"])
	if err != nil {
		return
	}
	defer download.Body.Close()
	sendAttachment(responseWriter, sendAttachmentInput{
		Title:         scans.Title,
		Format:        string(ebook.Format),
		ContentLength: download.ContentLength,
		Body:          download.Body,
	})
}

// sendAttachmentInput is an ebook to send, named after Title, with
// ContentLength -1 when unknown.
type sendAttachmentInput struct {
	Title         string
	Format        string
	ContentLength int64
	Body          io.Reader
}

// sendAttachment streams an ebook for the browser to save.
func sendAttachment(responseWriter http.ResponseWriter, input sendAttachmentInput) {
	fileName := fmt.Sprintf("%s.%s", fileNameOf(input.Title), input.Format)
	responseWriter.Header().Set("Content-Type", contentTypes[input.Format])
	responseWriter.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": fileName}))
	if input.ContentLength >= 0 {
		responseWriter.Header().Set("Content-Length", fmt.Sprint(input.ContentLength))
	}
	// Once the body has started, a failure can no longer be sent as an
	// error response.
	if _, err := io.Copy(responseWriter, input.Body); err != nil {
		logx.Warnf("sending %s: %v", fileName, err)
	}
}

// openPublicEbook starts downloading the ebook of a public domain work from
// the Internet Archive.
func (api *API) openPublicEbook(ctx context.Context, openLibraryID string) (openlibrary.Scans, internetarchive.Ebook, internetarchive.Download, error) {
	scans, err := api.openLibrary.PublicScans(openLibraryID)
	if err != nil {
		return openlibrary.Scans{}, internetarchive.Ebook{}, internetarchive.Download{}, err
	}
	ebook, err := api.internetArchive.FindEbook(scans.ArchiveIDs)
	if err != nil {
		return openlibrary.Scans{}, internetarchive.Ebook{}, internetarchive.Download{}, err
	}
	download, err := api.internetArchive.OpenEbook(ctx, ebook)
	if err != nil {
		return openlibrary.Scans{}, internetarchive.Ebook{}, internetarchive.Download{}, err
	}
	return scans, ebook, download, nil
}

// fileNameOf keeps a title to the characters safe in a file name.
func fileNameOf(title string) string {
	name := strings.Map(func(character rune) rune {
		if unicode.IsControl(character) || strings.ContainsRune(`/\:*?"<>|`, character) {
			return -1
		}
		return character
	}, title)
	name = strings.TrimSpace(name)
	if name == "" {
		return "book"
	}
	return name
}
