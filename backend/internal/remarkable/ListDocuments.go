package remarkable

import (
	"fmt"
	"strings"
	"time"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

// Record separators unlikely to appear in a document title/JSON payload.
// Each record is: \x01<uuid>\x02<metadata JSON>\x03<content JSON>\x04
const (
	sepRecordStart = "\x01"
	sepUUIDEnd     = "\x02"
	sepMetaEnd     = "\x03"
	sepRecordEnd   = "\x04"
)

// listScript, run in the documents directory, prints one such record per
// document (not folder) in the xochitl store, so a single SSH round trip is
// enough to fetch everything needed to classify and title every document on
// the tablet. The separators are printed with octal escapes, which POSIX
// printf guarantees, unlike \x hex escapes.
const listScript = `for f in *.metadata; do
  uuid="${f%.metadata}"
  printf '\001%s\002' "$uuid"
  cat "$f"
  printf '\003'
  [ -f "$uuid.content" ] && cat "$uuid.content"
  printf '\004'
done`

// ListDocuments SSHes into the given host and returns every document
// (excluding folders) found in the tablet's xochitl document store,
// classified as pdf/epub/notebook per Classify, with the page each was last
// left open on.
func (client *SSHClient) ListDocuments(tablet Tablet) (Listing, error) {
	connection, hostKey, err := client.dialKey(tablet)
	if err != nil {
		return Listing{}, err
	}
	defer connection.Close()
	output, err := run(connection, "cd '"+client.config.DocumentsDir+"' && "+listScript)
	if err != nil {
		return Listing{}, fmt.Errorf("listing the tablet's documents: %w", err)
	}
	listing, err := ParseListOutput(output)
	listing.HostKey = hostKey
	return listing, err
}

// ParseListOutput parses what listScript prints into documents and
// folders, skipping records it can't parse and ones the tablet has marked
// deleted, which it keeps only until they're gone from its cloud.
func ParseListOutput(output string) (Listing, error) {
	var docs []models.RemarkableDocument
	var folders []models.RemarkableFolder

	for _, record := range strings.Split(output, sepRecordStart) {
		if record == "" {
			continue
		}
		record = strings.TrimSuffix(record, sepRecordEnd)

		uuidAndRest := strings.SplitN(record, sepUUIDEnd, 2)
		if len(uuidAndRest) != 2 {
			continue
		}
		uuid := uuidAndRest[0]

		metaAndContent := strings.SplitN(uuidAndRest[1], sepMetaEnd, 2)
		if len(metaAndContent) != 2 {
			continue
		}

		meta, err := parseMetadata([]byte(metaAndContent[0]))
		if err != nil {
			continue // skip unparseable/partial records rather than failing the whole sync
		}
		if meta.Deleted {
			continue
		}
		if meta.Type == "CollectionType" {
			folders = append(folders, models.RemarkableFolder{UUID: uuid, Title: titleOf(meta, uuid), ParentUUID: meta.Parent})
			continue
		}
		if meta.Type != "DocumentType" {
			continue
		}

		c, _ := parseContent([]byte(metaAndContent[1]))

		lastModified, ok := parseMillis(meta.LastModified)
		if !ok {
			lastModified = time.Now().UTC()
		}

		document := models.RemarkableDocument{
			UUID:         uuid,
			Title:        titleOf(meta, uuid),
			ParentUUID:   meta.Parent,
			FileType:     Classify(c.FileType),
			LastModified: lastModified,
			BookTitle:    strings.TrimSpace(c.DocumentMetadata.Title),
			CoverPageID:  coverPageID(meta, c),
		}
		if len(c.DocumentMetadata.Authors) > 0 {
			document.BookAuthor = strings.TrimSpace(c.DocumentMetadata.Authors[0])
		}
		if position, ok := documentPosition(meta, c); ok {
			document.CurrentPage = &position.currentPage
			document.PageCount = &position.pageCount
			document.PositionUpdatedAt = &position.updatedAt
		}
		docs = append(docs, document)
	}

	return Listing{Documents: docs, Folders: folders}, nil
}

// titleOf is the name the tablet shows for an item, or its UUID if unnamed.
func titleOf(meta metadata, uuid string) string {
	if meta.VisibleName == "" {
		return uuid
	}
	return meta.VisibleName
}
