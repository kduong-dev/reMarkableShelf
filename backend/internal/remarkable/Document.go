package remarkable

import (
	"encoding/json"
	"strconv"
	"time"
)

// DefaultDocumentsDir is where documents live on the tablet's filesystem,
// per the setup guide in the repo README.
const DefaultDocumentsDir = "/home/root/.local/share/remarkable/xochitl"

// metadata mirrors the fields we need from a <uuid>.metadata file.
// LastOpened and LastModified are epoch millis as strings, with LastOpened
// "0" for a document never opened; LastOpenedPage counts from 0.
type metadata struct {
	VisibleName    string `json:"visibleName"`
	Type           string `json:"type"` // "DocumentType" or "CollectionType" (folder)
	LastModified   string `json:"lastModified"`
	LastOpened     string `json:"lastOpened"`
	LastOpenedPage *int   `json:"lastOpenedPage"`
	Parent         string `json:"parent"`
	Deleted        bool   `json:"deleted"`
}

// content mirrors the fields we need from a <uuid>.content file. fileType
// is the signal that distinguishes an imported PDF/EPUB from a native
// handwritten notebook — see Classify.go. documentMetadata holds the book's
// own title and authors when the tablet could read them. formatVersion 1
// lists page IDs in pages; formatVersion 2, which the tablet converts a
// document to once it's opened, lists them in cPages along with the open
// page. coverPageNumber is the page shown as the cover, -1 meaning the last
// one opened.
type content struct {
	FileType         string            `json:"fileType"`
	PageCount        int               `json:"pageCount"`
	Pages            []json.RawMessage `json:"pages"`
	CPages           *cPages           `json:"cPages"`
	CoverPageNumber  *int              `json:"coverPageNumber"`
	DocumentMetadata documentMetadata  `json:"documentMetadata"`
}

// pageIDs lists the document's pages in order, leaving out deleted ones.
func (c content) pageIDs() []string {
	var ids []string
	if c.CPages != nil {
		for _, page := range c.CPages.Pages {
			if !page.deleted() {
				ids = append(ids, page.ID)
			}
		}
		return ids
	}
	for _, raw := range c.Pages {
		var id string
		if json.Unmarshal(raw, &id) == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// documentMetadata is what the tablet read from the file itself, usually
// filled in for EPUBs and empty for PDFs.
type documentMetadata struct {
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
}

type cPages struct {
	LastOpened struct {
		Value string `json:"value"`
	} `json:"lastOpened"`
	Pages []cPage `json:"pages"`
}

type cPage struct {
	ID      string `json:"id"`
	Deleted *struct {
		Value int `json:"value"`
	} `json:"deleted"`
}

func (page cPage) deleted() bool {
	return page.Deleted != nil && page.Deleted.Value != 0
}

func parseMetadata(raw []byte) (metadata, error) {
	var m metadata
	err := json.Unmarshal(raw, &m)
	return m, err
}

func parseContent(raw []byte) (content, error) {
	if len(raw) == 0 {
		return content{}, nil
	}
	var c content
	err := json.Unmarshal(raw, &c)
	return c, err
}

// parseMillis parses an epoch-millis string, reporting false for a missing
// or zero timestamp.
func parseMillis(raw string) (time.Time, bool) {
	milliseconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || milliseconds <= 0 {
		return time.Time{}, false
	}
	return time.UnixMilli(milliseconds).UTC(), true
}

// position is where a document was last left open on the tablet.
type position struct {
	currentPage int
	pageCount   int
	updatedAt   time.Time
}

// documentPosition reads a document's last open page (counting from 1) and
// page count, reporting false for a document never opened or whose page
// can't be found. updatedAt is the later of the document's last open and
// last modification, since the tablet saves the page on leaving it, after
// lastOpened was stamped.
func documentPosition(meta metadata, c content) (position, bool) {
	openedAt, opened := parseMillis(meta.LastOpened)
	if !opened {
		return position{}, false
	}
	pageIDs := c.pageIDs()
	pageCount := c.PageCount
	if pageCount == 0 {
		pageCount = len(pageIDs)
	}
	currentPage := 0
	if c.CPages != nil && c.CPages.LastOpened.Value != "" {
		for index, id := range pageIDs {
			if id == c.CPages.LastOpened.Value {
				currentPage = index + 1
				break
			}
		}
	}
	if currentPage == 0 && meta.LastOpenedPage != nil {
		currentPage = *meta.LastOpenedPage + 1
	}
	if currentPage < 1 || pageCount < 1 || currentPage > pageCount {
		return position{}, false
	}
	updatedAt := openedAt
	if modifiedAt, ok := parseMillis(meta.LastModified); ok && modifiedAt.After(updatedAt) {
		updatedAt = modifiedAt
	}
	return position{currentPage: currentPage, pageCount: pageCount, updatedAt: updatedAt}, true
}

// coverPageID is the ID of the page the tablet shows as the document's
// cover, whose thumbnail is <uuid>.thumbnails/<id>.png, or "" if unknown.
func coverPageID(meta metadata, c content) string {
	pageIDs := c.pageIDs()
	coverPage := 0
	if c.CoverPageNumber != nil {
		coverPage = *c.CoverPageNumber
	}
	if coverPage == -1 {
		position, ok := documentPosition(meta, c)
		if !ok {
			coverPage = 0
		} else {
			coverPage = position.currentPage - 1
		}
	}
	if coverPage < 0 || coverPage >= len(pageIDs) {
		return ""
	}
	return pageIDs[coverPage]
}
