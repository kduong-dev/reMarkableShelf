package models_test

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

func TestScalePage(t *testing.T) {
	Convey("Given pages to move between page counts", t, func() {
		Convey("Then a page moves the same fraction through", func() {
			So(models.ScalePage(42, 171, 280), ShouldEqual, 69)
		})
		Convey("Then the last page stays the last page", func() {
			So(models.ScalePage(171, 171, 544), ShouldEqual, 544)
		})
		Convey("Then an early page never rounds down to 0", func() {
			So(models.ScalePage(1, 500, 100), ShouldEqual, 1)
		})
		Convey("Then page 0, not started, stays 0", func() {
			So(models.ScalePage(0, 171, 280), ShouldEqual, 0)
		})
	})
}
