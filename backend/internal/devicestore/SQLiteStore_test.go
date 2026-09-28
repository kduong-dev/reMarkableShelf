package devicestore_test

import (
	"path/filepath"
	"testing"

	"errors"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/database"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

func TestRenameDevice(t *testing.T) {
	Convey("Given a device named Test", t, func() {
		opened, err := database.Open(filepath.Join(t.TempDir(), "books.db"))
		So(err, ShouldBeNil)
		Reset(func() { _ = opened.Close() })
		devices := devicestore.NewSQLiteStore(opened)
		device, err := devices.Create(models.Device{Name: "Test", Host: "10.0.0.5"})
		So(err, ShouldBeNil)
		Convey("When renaming it", func() {
			renamed, err := devices.Rename(device.ID, "  Paper Pro  ")
			Convey("Then the trimmed name is saved", func() {
				So(err, ShouldBeNil)
				So(renamed.Name, ShouldEqual, "Paper Pro")
				stored, err := devices.Get(device.ID)
				So(err, ShouldBeNil)
				So(stored.Name, ShouldEqual, "Paper Pro")
			})
		})
		Convey("When renaming it to blank", func() {
			_, err := devices.Rename(device.ID, "   ")
			Convey("Then it returns ErrDeviceNameRequired", func() {
				So(errors.Is(err, devicestore.ErrDeviceNameRequired), ShouldBeTrue)
			})
		})
		Convey("When renaming a device that doesn't exist", func() {
			_, err := devices.Rename("missing", "Paper Pro")
			Convey("Then it returns ErrDeviceNotFound", func() {
				So(errors.Is(err, devicestore.ErrDeviceNotFound), ShouldBeTrue)
			})
		})
	})
}
