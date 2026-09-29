package remarkable

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"golang.org/x/crypto/ssh"
)

// newMetadata is the <uuid>.metadata of a document the tablet has never
// opened, or of a folder, inside the folder Parent ("" for the top of the
// library).
type newMetadata struct {
	Deleted          bool   `json:"deleted"`
	LastModified     string `json:"lastModified"`
	LastOpened       string `json:"lastOpened"`
	LastOpenedPage   int    `json:"lastOpenedPage"`
	MetadataModified bool   `json:"metadatamodified"`
	Modified         bool   `json:"modified"`
	Parent           string `json:"parent"`
	Pinned           bool   `json:"pinned"`
	Synced           bool   `json:"synced"`
	Type             string `json:"type"`
	Version          int    `json:"version"`
	VisibleName      string `json:"visibleName"`
}

// newContent is the <uuid>.content of a document not yet opened, which the
// tablet fills in when it first opens it.
type newContent struct {
	CoverPageNumber int               `json:"coverPageNumber"`
	ExtraMetadata   map[string]string `json:"extraMetadata"`
	FileType        models.FileType   `json:"fileType"`
	FontName        string            `json:"fontName"`
	LineHeight      int               `json:"lineHeight"`
	Margins         int               `json:"margins"`
	Orientation     string            `json:"orientation"`
	PageCount       int               `json:"pageCount"`
	TextScale       int               `json:"textScale"`
}

// CopyDocument writes the file, then its .content, then its .metadata,
// since the tablet takes a document to exist once its .metadata does. Each
// is written beside its place and moved there, so no half-written file is
// ever seen.
func (client *SSHClient) CopyDocument(tablet Tablet, input CopyDocumentInput) error {
	if !idPattern.MatchString(input.UUID) {
		return fmt.Errorf("%w: uuid %q", ErrInvalidDocument, input.UUID)
	}
	if input.FileType != models.FileTypeEPUB && input.FileType != models.FileTypePDF {
		return fmt.Errorf("%w: file type %q", ErrInvalidDocument, input.FileType)
	}
	connection, _, err := client.dialKey(tablet)
	if err != nil {
		return err
	}
	defer connection.Close()
	now := strconv.FormatInt(time.Now().UnixMilli(), 10)
	content := fatal.UnlessMarshal(newContent{
		ExtraMetadata: map[string]string{},
		FileType:      input.FileType,
		LineHeight:    -1,
		Margins:       100,
		Orientation:   "portrait",
		TextScale:     1,
	})
	metadata := fatal.UnlessMarshal(newMetadata{
		LastModified: now,
		LastOpened:   "0",
		Parent:       input.ParentUUID,
		Type:         "DocumentType",
		VisibleName:  input.Title,
	})
	return client.writeItem(connection, input.UUID, []itemFile{
		{string(input.FileType), input.Body},
		{"content", bytes.NewReader(content)},
		{"metadata", bytes.NewReader(metadata)},
	})
}

// CreateFolder writes the folder's .content, then its .metadata, which
// makes it exist.
func (client *SSHClient) CreateFolder(tablet Tablet, folder models.RemarkableFolder) error {
	if !idPattern.MatchString(folder.UUID) {
		return fmt.Errorf("%w: uuid %q", ErrInvalidDocument, folder.UUID)
	}
	connection, _, err := client.dialKey(tablet)
	if err != nil {
		return err
	}
	defer connection.Close()
	metadata := fatal.UnlessMarshal(newMetadata{
		LastModified: strconv.FormatInt(time.Now().UnixMilli(), 10),
		LastOpened:   "0",
		Parent:       folder.ParentUUID,
		Type:         "CollectionType",
		VisibleName:  folder.Title,
	})
	return client.writeItem(connection, folder.UUID, []itemFile{
		{"content", strings.NewReader(`{"tags":[]}`)},
		{"metadata", bytes.NewReader(metadata)},
	})
}

// itemFile is one of the files making up a document or folder, named
// <uuid>.<extension>.
type itemFile struct {
	extension string
	body      io.Reader
}

// writeItem writes an item's files in order, so its .metadata, written
// last, only appears once the rest are in place.
func (client *SSHClient) writeItem(connection *ssh.Client, uuid string, files []itemFile) error {
	for _, file := range files {
		if err := client.writeFile(connection, uuid+"."+file.extension, file.body); err != nil {
			return fmt.Errorf("copying %s.%s to the tablet: %w", uuid, file.extension, err)
		}
	}
	return nil
}

// writeFile writes body to name in the documents directory. name is built
// from a checked UUID, so it's safe in the script.
func (client *SSHClient) writeFile(connection *ssh.Client, name string, body io.Reader) error {
	path := client.config.DocumentsDir + "/" + name
	_, err := runWithInput(connection, fmt.Sprintf("cat > '%[1]s.part' && mv '%[1]s.part' '%[1]s'", path), body)
	return err
}

// restartAppScript restarts xochitl, retrying once, since systemd cancels
// a restart when another job for it arrives first, as when the tablet
// starts to sleep. If both fail it still starts xochitl, so the tablet is
// never left without its reading app.
const restartAppScript = `systemctl restart xochitl || { sleep 3; systemctl restart xochitl; } || { systemctl start xochitl; exit 1; }`

func (client *SSHClient) RestartApp(tablet Tablet) error {
	connection, _, err := client.dialKey(tablet)
	if err != nil {
		return err
	}
	defer connection.Close()
	if _, err := run(connection, restartAppScript); err != nil {
		return fmt.Errorf("%w: %w", ErrRestartFailed, err)
	}
	return nil
}
