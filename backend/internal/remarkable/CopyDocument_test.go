package remarkable_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/remarkable"
)

// pairedTablet starts a fake tablet that already trusts client's key.
func pairedTablet(t *testing.T) (*fakeTablet, *remarkable.SSHClient, remarkable.Tablet, string) {
	tablet := startFakeTablet(t, "secret")
	documentsDir := filepath.Join(tablet.home, "xochitl")
	So(os.MkdirAll(documentsDir, 0o755), ShouldBeNil)
	client := newClientFor(t, tablet.port, documentsDir)
	hostKey, err := client.Pair(remarkable.Tablet{Host: tablet.host}, "secret")
	So(err, ShouldBeNil)
	return tablet, client, remarkable.Tablet{Host: tablet.host, HostKey: hostKey}, documentsDir
}

func TestCopyDocument(t *testing.T) {
	Convey("Given a paired tablet", t, func() {
		_, client, target, documentsDir := pairedTablet(t)
		uuid := "0b8e2f7a-5c1d-4e3b-9a6f-2d4c8e1b7a90"
		Convey("When copying an EPUB to it", func() {
			err := client.CopyDocument(target, remarkable.CopyDocumentInput{
				UUID:     uuid,
				Title:    `Pride & "Prejudice"`,
				FileType: models.FileTypeEPUB,
				Body:     strings.NewReader("epub contents"),
			})
			So(err, ShouldBeNil)
			Convey("Then the file lands in the documents directory under its UUID", func() {
				contents, err := os.ReadFile(filepath.Join(documentsDir, uuid+".epub"))
				So(err, ShouldBeNil)
				So(string(contents), ShouldEqual, "epub contents")
			})
			Convey("Then its metadata names it, as a document never opened", func() {
				raw, err := os.ReadFile(filepath.Join(documentsDir, uuid+".metadata"))
				So(err, ShouldBeNil)
				var metadata map[string]any
				So(json.Unmarshal(raw, &metadata), ShouldBeNil)
				So(metadata["visibleName"], ShouldEqual, `Pride & "Prejudice"`)
				So(metadata["type"], ShouldEqual, "DocumentType")
				So(metadata["lastOpened"], ShouldEqual, "0")
				So(metadata["parent"], ShouldEqual, "")
			})
			Convey("Then its content gives its file type", func() {
				raw, err := os.ReadFile(filepath.Join(documentsDir, uuid+".content"))
				So(err, ShouldBeNil)
				var content map[string]any
				So(json.Unmarshal(raw, &content), ShouldBeNil)
				So(content["fileType"], ShouldEqual, "epub")
			})
			Convey("Then no partly written files are left", func() {
				leftovers, _ := filepath.Glob(filepath.Join(documentsDir, "*.part"))
				So(leftovers, ShouldBeEmpty)
			})
			Convey("Then listing the tablet finds it", func() {
				listing, err := client.ListDocuments(target)
				So(err, ShouldBeNil)
				So(listing.Documents, ShouldHaveLength, 1)
				So(listing.Documents[0].UUID, ShouldEqual, uuid)
				So(listing.Documents[0].Title, ShouldEqual, `Pride & "Prejudice"`)
				So(listing.Documents[0].FileType, ShouldEqual, models.FileTypeEPUB)
			})
		})
		Convey("When copying with a UUID that could escape the directory", func() {
			err := client.CopyDocument(target, remarkable.CopyDocumentInput{
				UUID: "../x", FileType: models.FileTypePDF, Body: strings.NewReader("pdf"),
			})
			Convey("Then it's refused without writing anything", func() {
				So(errors.Is(err, remarkable.ErrInvalidDocument), ShouldBeTrue)
				entries, _ := os.ReadDir(documentsDir)
				So(entries, ShouldBeEmpty)
			})
		})
	})
}

func TestCreateFolder(t *testing.T) {
	Convey("Given a paired tablet", t, func() {
		_, client, target, documentsDir := pairedTablet(t)
		Convey("When creating a folder and copying a PDF into it", func() {
			So(client.CreateFolder(target, models.RemarkableFolder{UUID: "5d1e0c3a-d00d", Title: "Downloads"}), ShouldBeNil)
			So(client.CopyDocument(target, remarkable.CopyDocumentInput{
				UUID:       "9f2b7e41-e44a",
				Title:      "Emma",
				ParentUUID: "5d1e0c3a-d00d",
				FileType:   models.FileTypePDF,
				Body:       strings.NewReader("%PDF-1.7"),
			}), ShouldBeNil)
			Convey("Then the folder is a collection with no file of its own", func() {
				raw, err := os.ReadFile(filepath.Join(documentsDir, "5d1e0c3a-d00d.metadata"))
				So(err, ShouldBeNil)
				var metadata map[string]any
				So(json.Unmarshal(raw, &metadata), ShouldBeNil)
				So(metadata["type"], ShouldEqual, "CollectionType")
				So(metadata["visibleName"], ShouldEqual, "Downloads")
				_, err = os.Stat(filepath.Join(documentsDir, "5d1e0c3a-d00d.content"))
				So(err, ShouldBeNil)
			})
			Convey("Then listing the tablet finds the folder with the PDF inside", func() {
				listing, err := client.ListDocuments(target)
				So(err, ShouldBeNil)
				So(listing.Folders, ShouldResemble, []models.RemarkableFolder{{UUID: "5d1e0c3a-d00d", Title: "Downloads"}})
				So(listing.Documents, ShouldHaveLength, 1)
				So(listing.Documents[0].ParentUUID, ShouldEqual, "5d1e0c3a-d00d")
			})
		})
	})
}

// fakeSystemctl puts a stand-in systemctl first on the PATH that logs its
// arguments to the returned file and fails the first failures calls, so
// tests never touch the machine's real services.
func fakeSystemctl(t *testing.T, home string, failures int) string {
	bin := t.TempDir()
	log := filepath.Join(home, "systemctl.log")
	count := filepath.Join(home, "systemctl.count")
	script := fmt.Sprintf(`#!/bin/sh
echo "$@" >> '%s'
calls=$(( $(cat '%s' 2>/dev/null || echo 0) + 1 ))
echo $calls > '%s'
[ $calls -gt %d ] || { echo "Job for xochitl.service canceled." >&2; exit 1; }
`, log, count, count, failures)
	writeFile(filepath.Join(bin, "systemctl"), script)
	So(os.Chmod(filepath.Join(bin, "systemctl"), 0o755), ShouldBeNil)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func TestRestartApp(t *testing.T) {
	Convey("Given a paired tablet whose systemctl works", t, func() {
		tablet, client, target, _ := pairedTablet(t)
		log := fakeSystemctl(t, tablet.home, 0)
		Convey("When restarting its reading app", func() {
			err := client.RestartApp(target)
			Convey("Then it restarts xochitl once", func() {
				So(err, ShouldBeNil)
				calls, err := os.ReadFile(log)
				So(err, ShouldBeNil)
				So(string(calls), ShouldEqual, "restart xochitl\n")
			})
		})
	})
	Convey("Given a paired tablet that cancels the first restart", t, func() {
		tablet, client, target, _ := pairedTablet(t)
		log := fakeSystemctl(t, tablet.home, 1)
		Convey("When restarting its reading app", func() {
			err := client.RestartApp(target)
			Convey("Then it retries and succeeds", func() {
				So(err, ShouldBeNil)
				calls, err := os.ReadFile(log)
				So(err, ShouldBeNil)
				So(string(calls), ShouldEqual, "restart xochitl\nrestart xochitl\n")
			})
		})
	})
	Convey("Given a paired tablet that cancels every restart", t, func() {
		tablet, client, target, _ := pairedTablet(t)
		log := fakeSystemctl(t, tablet.home, 2)
		Convey("When restarting its reading app", func() {
			err := client.RestartApp(target)
			Convey("Then it makes sure xochitl is started, and returns ErrRestartFailed", func() {
				So(errors.Is(err, remarkable.ErrRestartFailed), ShouldBeTrue)
				calls, err := os.ReadFile(log)
				So(err, ShouldBeNil)
				So(string(calls), ShouldEqual, "restart xochitl\nrestart xochitl\nstart xochitl\n")
			})
		})
	})
}
