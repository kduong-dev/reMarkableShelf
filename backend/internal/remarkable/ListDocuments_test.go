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
			listing, err := remarkable.ParseListOutput(output)
			documents := listing.Documents
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
			listing, err := remarkable.ParseListOutput(output)
			documents := listing.Documents
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
			listing, err := remarkable.ParseListOutput(output)
			documents := listing.Documents
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
	Convey("Given documents whose cover pages differ", t, func() {
		output := record("first-page", metadataOpenedAt("0", 0), formatVersion1Content) +
			record("deleted-first", `{"type": "DocumentType", "visibleName": "a.epub"}`, `{
				"fileType": "epub",
				"cPages": {"pages": [{"id": "gone", "deleted": {"value": 1}}, {"id": "page-b"}]}
			}`) +
			record("last-opened", `{"type": "DocumentType", "visibleName": "b.epub", "lastOpened": "1759000000000"}`, `{
				"fileType": "epub",
				"coverPageNumber": -1,
				"cPages": {"lastOpened": {"value": "page-y"}, "pages": [{"id": "page-x"}, {"id": "page-y"}]}
			}`)
		Convey("When parsing the listing", func() {
			listing, err := remarkable.ParseListOutput(output)
			documents := listing.Documents
			So(err, ShouldBeNil)
			coverByUUID := map[string]string{}
			for _, document := range documents {
				coverByUUID[document.UUID] = document.CoverPageID
			}
			Convey("Then a formatVersion 1 document's cover is its first listed page", func() {
				So(coverByUUID["first-page"], ShouldEqual, "f3df5538-f65e-44f2-a83a-01f4cd91289f")
			})
			Convey("Then a formatVersion 2 document skips deleted pages", func() {
				So(coverByUUID["deleted-first"], ShouldEqual, "page-b")
			})
			Convey("Then coverPageNumber -1 means the page last opened", func() {
				So(coverByUUID["last-opened"], ShouldEqual, "page-y")
			})
		})
	})
	Convey("Given folders, a document inside one, and items in the trash or deleted", t, func() {
		output := record("books", `{"type": "CollectionType", "visibleName": "Books", "parent": ""}`, "") +
			record("classics", `{"type": "CollectionType", "visibleName": "Classics", "parent": "books"}`, "") +
			record("old", `{"type": "CollectionType", "visibleName": "Old", "parent": "trash"}`, "") +
			record("086daeda", `{"type": "DocumentType", "visibleName": "Dune.pdf", "parent": "classics"}`, formatVersion1Content) +
			record("top", `{"type": "DocumentType", "visibleName": "Quick sheets"}`, "") +
			record("binned", `{"type": "DocumentType", "visibleName": "Draft", "parent": "trash"}`, "") +
			record("gone", `{"type": "DocumentType", "visibleName": "Gone", "deleted": true}`, "")
		Convey("When parsing the listing", func() {
			listing, err := remarkable.ParseListOutput(output)
			So(err, ShouldBeNil)
			Convey("Then every folder is returned inside its parent, including trashed ones", func() {
				So(listing.Folders, ShouldResemble, []models.RemarkableFolder{
					{UUID: "books", Title: "Books"},
					{UUID: "classics", Title: "Classics", ParentUUID: "books"},
					{UUID: "old", Title: "Old", ParentUUID: models.TrashFolderUUID},
				})
			})
			Convey("Then each document keeps its folder, and deleted ones are left out", func() {
				parents := map[string]string{}
				for _, document := range listing.Documents {
					parents[document.UUID] = document.ParentUUID
				}
				So(parents, ShouldResemble, map[string]string{"086daeda": "classics", "top": "", "binned": models.TrashFolderUUID})
			})
		})
	})
}

func TestParseEmbeddedMetadata(t *testing.T) {
	Convey("Given EPUBs whose embedded title and author are placeholders or written surname first", t, func() {
		epub := func(uuid, title, author string) string {
			return record(uuid, `{"type": "DocumentType", "visibleName": "Book"}`,
				fmt.Sprintf(`{"fileType": "epub", "documentMetadata": {"title": %q, "authors": [%q]}}`, title, author))
		}
		output := epub("clean-code", "0d66f70d-fa40-48b7-b3c6-a83f6d3a4b02", "Unknown") +
			epub("microservices", "Building Microservices", "Newman, Sam;") +
			epub("nations", "  Why Nations Fail ", "Daron  Acemoglu") +
			epub("urn", "urn:uuid:0d66f70d-fa40-48b7-b3c6-a83f6d3a4b02", "unknown author")
		Convey("When parsing the listing", func() {
			listing, err := remarkable.ParseListOutput(output)
			So(err, ShouldBeNil)
			metadata := map[string][2]string{}
			for _, document := range listing.Documents {
				metadata[document.UUID] = [2]string{document.BookTitle, document.BookAuthor}
			}
			Convey("Then placeholders are dropped, and names are tidied and put first name first", func() {
				So(metadata, ShouldResemble, map[string][2]string{
					"clean-code":    {"", ""},
					"microservices": {"Building Microservices", "Sam Newman"},
					"nations":       {"Why Nations Fail", "Daron Acemoglu"},
					"urn":           {"", ""},
				})
			})
		})
	})
}
