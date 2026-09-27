package store_test

import (
	"path/filepath"
	"testing"

	"github.com/ansel1/merry"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/store"
)

func TestRenameDevice(t *testing.T) {
	Convey("Given a device named Test", t, func() {
		opened, err := store.Open(filepath.Join(t.TempDir(), "books.db"))
		So(err, ShouldBeNil)
		Reset(func() { _ = opened.Close() })
		device, err := opened.CreateDevice(models.Device{Name: "Test", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		Convey("When renaming it", func() {
			renamed, err := opened.RenameDevice(device.ID, "  Paper Pro  ")
			Convey("Then the trimmed name is saved", func() {
				So(err, ShouldBeNil)
				So(renamed.Name, ShouldEqual, "Paper Pro")
				stored, err := opened.GetDevice(device.ID)
				So(err, ShouldBeNil)
				So(stored.Name, ShouldEqual, "Paper Pro")
			})
		})
		Convey("When renaming it to blank", func() {
			_, err := opened.RenameDevice(device.ID, "   ")
			Convey("Then it returns ErrDeviceNameRequired", func() {
				So(merry.Is(err, store.ErrDeviceNameRequired), ShouldBeTrue)
			})
		})
		Convey("When renaming a device that doesn't exist", func() {
			_, err := opened.RenameDevice("missing", "Paper Pro")
			Convey("Then it responds not found", func() {
				So(merry.HTTPCode(err), ShouldEqual, 404)
			})
		})
	})
}
