package remarkable_test

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/remarkable"
)

func record(uuid, metadata, content string) string {
	return "\x01" + uuid + "\x02" + metadata + "\x03" + content + "\x04"
}

// formatVersion1Metadata is trimmed from a real Paper Pro document: a PDF
// that had never been opened.
const formatVersion1Metadata = `{
    "createdTime": "1741415702915",
    "lastModified": "1741415702914",
    "lastOpened": "%s",
    "lastOpenedPage": %d,
    "type": "DocumentType",
    "visibleName": "Dale Carnegie - How to Win Friends and Influence People.pdf"
}`

const formatVersion1Content = `{
    "fileType": "pdf",
    "formatVersion": 1,
    "originalPageCount": 171,
    "pageCount": 171,
    "pages": ["f3df5538-f65e-44f2-a83a-01f4cd91289f", "39960184-8156-40ff-a4a9-110d3e2f36ef"]
}`

func metadataOpenedAt(lastOpened string, lastOpenedPage int) string {
	return fmt.Sprintf(formatVersion1Metadata, lastOpened, lastOpenedPage)
}

func TestParseListOutput(t *testing.T) {
	Convey("Given a formatVersion 1 PDF that has never been opened", t, func() {
		output := record("086daeda", metadataOpenedAt("0", 0), formatVersion1Content)
		Convey("When parsing the listing", func() {
			documents, err := remarkable.ParseListOutput(output)
			Convey("Then it has no reading position", func() {
				So(err, ShouldBeNil)
				So(documents, ShouldHaveLength, 1)
				So(documents[0].FileType, ShouldEqual, models.FileTypePDF)
				So(documents[0].CurrentPage, ShouldBeNil)
				So(documents[0].PageCount, ShouldBeNil)
			})
		})
	})
	Convey("Given a formatVersion 1 PDF left open on its 42nd page", t, func() {
		output := record("086daeda", metadataOpenedAt("1759000000000", 41), formatVersion1Content)
		Convey("When parsing the listing", func() {
			documents, err := remarkable.ParseListOutput(output)
			Convey("Then its position counts from 1, with the later of last opened and last modified", func() {
				So(err, ShouldBeNil)
				So(*documents[0].CurrentPage, ShouldEqual, 42)
				So(*documents[0].PageCount, ShouldEqual, 171)
				So(*documents[0].PositionUpdatedAt, ShouldEqual, time.UnixMilli(1759000000000).UTC())
			})
		})
	})
	Convey("Given a formatVersion 2 EPUB whose open page is recorded in cPages", t, func() {
		output := record("5b1c", `{"type": "DocumentType", "visibleName": "piranesi_clarke.epub", "lastOpened": "1759000000000", "lastModified": "1759000500000"}`, `{
			"fileType": "epub",
			"formatVersion": 2,
			"documentMetadata": {"title": "Piranesi", "authors": ["Susanna Clarke"]},
			"pageCount": 3,
			"cPages": {
				"lastOpened": {"timestamp": "1:2", "value": "page-c"},
				"pages": [
					{"id": "page-a"},
					{"id": "page-deleted", "deleted": {"timestamp": "1:3", "value": 1}},
					{"id": "page-b"},
					{"id": "page-c"}
				]
			}
		}`)
		Convey("When parsing the listing", func() {
			documents, err := remarkable.ParseListOutput(output)
			Convey("Then the page is its place among the pages that aren't deleted", func() {
				So(err, ShouldBeNil)
				So(documents[0].FileType, ShouldEqual, models.FileTypeEPUB)
				So(*documents[0].CurrentPage, ShouldEqual, 3)
				So(*documents[0].PageCount, ShouldEqual, 3)
				So(*documents[0].PositionUpdatedAt, ShouldEqual, time.UnixMilli(1759000500000).UTC())
			})
			Convey("Then the book's own title and author are kept beside its file name", func() {
				So(documents[0].Title, ShouldEqual, "piranesi_clarke.epub")
				So(documents[0].BookTitle, ShouldEqual, "Piranesi")
				So(documents[0].BookAuthor, ShouldEqual, "Susanna Clarke")
			})
		})
	})
	Convey("Given a folder among the documents", t, func() {
		output := record("folder", `{"type": "CollectionType", "visibleName": "Books"}`, "") +
			record("086daeda", metadataOpenedAt("0", 0), formatVersion1Content)
		Convey("When parsing the listing", func() {
			documents, err := remarkable.ParseListOutput(output)
			Convey("Then only the document is returned", func() {
				So(err, ShouldBeNil)
				So(documents, ShouldHaveLength, 1)
				So(documents[0].UUID, ShouldEqual, "086daeda")
			})
		})
	})
}
