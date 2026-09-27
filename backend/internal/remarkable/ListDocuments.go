package remarkable

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/ansel1/merry"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"golang.org/x/crypto/ssh"
)

// Record separators unlikely to appear in a document title/JSON payload.
// Each record is: \x01<uuid>\x02<metadata JSON>\x03<content JSON>\x04
const (
	sepRecordStart = "\x01"
	sepUUIDEnd     = "\x02"
	sepMetaEnd     = "\x03"
	sepRecordEnd   = "\x04"
)

// listScript prints one such record per document (not folder) in the
// xochitl store, so a single SSH round trip is enough to fetch everything
// needed to classify and title every document on the tablet.
const listScript = `cd '` + xochitlDir + `' && for f in *.metadata; do
  uuid="${f%.metadata}"
  printf '\x01%s\x02' "$uuid"
  cat "$f"
  printf '\x03'
  [ -f "$uuid.content" ] && cat "$uuid.content"
  printf '\x04'
done`

// ListDocuments SSHes into the given host and returns every document
// (excluding folders) found in the tablet's xochitl document store,
// classified as pdf/epub/notebook per Classify, with the page each was last
// left open on.
func (client *SSHClient) ListDocuments(host string) ([]models.RemarkableDocument, error) {
	addr := fmt.Sprintf("%s:%s", host, client.config.Port)
	sshConfig := &ssh.ClientConfig{
		User:            client.config.User,
		Auth:            []ssh.AuthMethod{ssh.Password(client.config.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	connection, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, merry.Wrap(err).WithUserMessagef("connecting to tablet at %s", addr)
	}
	defer connection.Close()

	session, err := connection.NewSession()
	if err != nil {
		return nil, merry.Wrap(err).WithUserMessage("opening SSH session")
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	if err := session.Run(listScript); err != nil {
		return nil, merry.Wrap(err).WithUserMessagef("listing documents: %s", stderr.String())
	}

	return ParseListOutput(stdout.String())
}

// ParseListOutput parses what listScript prints into documents, skipping
// folders and records it can't parse.
func ParseListOutput(output string) ([]models.RemarkableDocument, error) {
	var docs []models.RemarkableDocument

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
		if meta.Type != "DocumentType" {
			continue // folders (CollectionType) aren't documents
		}

		c, _ := parseContent([]byte(metaAndContent[1]))

		lastModified, ok := parseMillis(meta.LastModified)
		if !ok {
			lastModified = time.Now().UTC()
		}

		title := meta.VisibleName
		if title == "" {
			title = uuid
		}

		document := models.RemarkableDocument{
			UUID:         uuid,
			Title:        title,
			FileType:     Classify(c.FileType),
			LastModified: lastModified,
		}
		if position, ok := documentPosition(meta, c); ok {
			document.CurrentPage = &position.currentPage
			document.PageCount = &position.pageCount
			document.PositionUpdatedAt = &position.updatedAt
		}
		docs = append(docs, document)
	}

	return docs, nil
}
