package sourcestore_test

import (
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/sourcestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/storetest"
)

func TestSQLiteStore(t *testing.T) {
	Convey("Given a new library", t, func() {
		store := storetest.OpenStores(t).SourceStore
		ids := func() []string {
			sources, err := store.List()
			So(err, ShouldBeNil)
			listed := []string{}
			for _, source := range sources {
				listed = append(listed, source.ID)
			}
			return listed
		}
		Convey("When listing its sources", func() {
			sources, err := store.List()
			Convey("Then it has only the built-in Internet Archive, enabled", func() {
				So(err, ShouldBeNil)
				So(sources, ShouldHaveLength, 1)
				So(sources[0].ID, ShouldEqual, models.BuiltInEbookSourceID)
				So(sources[0].Kind, ShouldEqual, models.EbookSourceKindInternetArchive)
				So(sources[0].Enabled, ShouldBeTrue)
			})
		})
		Convey("When adding a plugin and a catalog", func() {
			plugin, err := store.Create(models.EbookSource{Kind: models.EbookSourceKindPlugin, Name: "Folder", URL: "http://folder-source:8090"})
			So(err, ShouldBeNil)
			catalog, err := store.Create(models.EbookSource{Kind: models.EbookSourceKindOPDS, Name: "Project Gutenberg", URL: "https://m.gutenberg.org/ebooks.opds/", SearchURL: "https://m.gutenberg.org/search?q={searchTerms}"})
			So(err, ShouldBeNil)
			Convey("Then they're enabled and tried after the built-in source, in the order added", func() {
				So(plugin.Enabled, ShouldBeTrue)
				So(ids(), ShouldResemble, []string{models.BuiltInEbookSourceID, plugin.ID, catalog.ID})
			})
			Convey("Then the catalog keeps its search template", func() {
				stored, err := store.Get(catalog.ID)
				So(err, ShouldBeNil)
				So(stored.SearchURL, ShouldEqual, "https://m.gutenberg.org/search?q={searchTerms}")
			})
			Convey("And adding the plugin again", func() {
				_, err := store.Create(models.EbookSource{Kind: models.EbookSourceKindPlugin, Name: "Folder", URL: "http://folder-source:8090"})
				Convey("Then it's refused", func() {
					So(errors.Is(err, sourcestore.ErrSourceExists), ShouldBeTrue)
				})
			})
			Convey("And reordering them", func() {
				So(store.Reorder([]string{catalog.ID, models.BuiltInEbookSourceID, plugin.ID}), ShouldBeNil)
				Convey("Then they're listed in that order", func() {
					So(ids(), ShouldResemble, []string{catalog.ID, models.BuiltInEbookSourceID, plugin.ID})
				})
			})
			Convey("And reordering without one of them", func() {
				err := store.Reorder([]string{catalog.ID, plugin.ID})
				Convey("Then it's refused and the order kept", func() {
					So(errors.Is(err, sourcestore.ErrInvalidOrder), ShouldBeTrue)
					So(ids(), ShouldResemble, []string{models.BuiltInEbookSourceID, plugin.ID, catalog.ID})
				})
			})
			Convey("And disabling the plugin", func() {
				disabled, err := store.SetEnabled(plugin.ID, false)
				Convey("Then it's kept, disabled", func() {
					So(err, ShouldBeNil)
					So(disabled.Enabled, ShouldBeFalse)
				})
			})
			Convey("And removing the plugin", func() {
				So(store.Delete(plugin.ID), ShouldBeNil)
				Convey("Then it's gone", func() {
					So(ids(), ShouldResemble, []string{models.BuiltInEbookSourceID, catalog.ID})
				})
			})
		})
		Convey("When removing the built-in source", func() {
			err := store.Delete(models.BuiltInEbookSourceID)
			Convey("Then it's refused", func() {
				So(errors.Is(err, sourcestore.ErrBuiltInSource), ShouldBeTrue)
			})
		})
		Convey("When getting a source that doesn't exist", func() {
			_, err := store.Get("missing")
			Convey("Then it returns ErrSourceNotFound", func() {
				So(errors.Is(err, sourcestore.ErrSourceNotFound), ShouldBeTrue)
			})
		})
	})
}
