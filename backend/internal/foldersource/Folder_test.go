package foldersource_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/foldersource"
	"github.com/kduong-dev/reMarkableShelf/backend/pkg/sourceplugin"
)

func TestFolder(t *testing.T) {
	Convey("Given a folder of ebooks and other files", t, func() {
		directory := t.TempDir()
		write := func(name, contents string) {
			So(os.MkdirAll(filepath.Dir(filepath.Join(directory, name)), 0o755), ShouldBeNil)
			So(os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o644), ShouldBeNil)
		}
		write("Robert C. Martin - Clean Code.epub", "clean code")
		write("purchases/Clean Architecture.pdf", "clean architecture")
		write("Clean Code notes.txt", "notes")
		write("Code Complete.epub", "code complete")
		folder := foldersource.New("Purchases", directory)
		ctx := context.Background()
		Convey("When finding a book", func() {
			ebooks, err := folder.FindEbooks(ctx, sourceplugin.FindRequest{Title: "Clean Code: A Handbook of Agile Software Craftsmanship"})
			Convey("Then only EPUBs and PDFs naming every word of its main title are found", func() {
				So(err, ShouldBeNil)
				So(ebooks, ShouldResemble, []sourceplugin.Ebook{{
					ID: "Robert C. Martin - Clean Code.epub", Title: "Robert C. Martin - Clean Code", Format: sourceplugin.FormatEPUB,
					Size: 10, Description: "Robert C. Martin - Clean Code.epub",
				}})
			})
		})
		Convey("When finding a book in a subfolder", func() {
			ebooks, err := folder.FindEbooks(ctx, sourceplugin.FindRequest{Title: "Clean Architecture"})
			Convey("Then its ID is its path from the folder", func() {
				So(err, ShouldBeNil)
				So(ebooks, ShouldHaveLength, 1)
				So(ebooks[0].ID, ShouldEqual, "purchases/Clean Architecture.pdf")
			})
			Convey("And opening it", func() {
				content, err := folder.OpenEbook(ctx, ebooks[0].ID)
				So(err, ShouldBeNil)
				defer content.Body.Close()
				contents, _ := io.ReadAll(content.Body)
				Convey("Then it returns the file", func() {
					So(string(contents), ShouldEqual, "clean architecture")
					So(content.Format, ShouldEqual, sourceplugin.FormatPDF)
				})
			})
		})
		Convey("When opening a path out of the folder, or a file that isn't an ebook", func() {
			for _, id := range []string{"../secret.epub", "/etc/passwd", "Clean Code notes.txt", "missing.epub"} {
				_, err := folder.OpenEbook(ctx, id)
				So(errors.Is(err, sourceplugin.ErrEbookNotFound), ShouldBeTrue)
			}
		})
	})
}
