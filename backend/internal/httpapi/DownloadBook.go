package httpapi

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/logx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/internetarchive"
)

var contentTypes = map[internetarchive.Format]string{
	internetarchive.FormatEPUB: "application/epub+zip",
	internetarchive.FormatPDF:  "application/pdf",
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
	scans, err := api.openLibrary.PublicScans(mux.Vars(request)["openLibraryId"])
	if err != nil {
		return
	}
	ebook, err := api.internetArchive.FindEbook(scans.ArchiveIDs)
	if err != nil {
		return
	}
	download, err := api.internetArchive.OpenEbook(request.Context(), ebook)
	if err != nil {
		return
	}
	defer download.Body.Close()
	fileName := fmt.Sprintf("%s.%s", fileNameOf(scans.Title), ebook.Format)
	responseWriter.Header().Set("Content-Type", contentTypes[ebook.Format])
	responseWriter.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": fileName}))
	if download.ContentLength >= 0 {
		responseWriter.Header().Set("Content-Length", fmt.Sprint(download.ContentLength))
	}
	// Once the body has started, a failure can no longer be sent as an
	// error response.
	if _, copyErr := io.Copy(responseWriter, download.Body); copyErr != nil {
		logx.Warnf("downloading %s/%s: %v", ebook.Identifier, ebook.FileName, copyErr)
	}
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
