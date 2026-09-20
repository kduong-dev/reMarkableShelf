package httpx_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ansel1/merry"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/kqvd/reMarkableShelf/backend/internal/httpx"
)

func TestSendErrorResponse(t *testing.T) {
	Convey("Given a merrified error with an HTTP code and user message", t, func() {
		err := merry.New("book not found").WithHTTPCode(http.StatusNotFound).WithUserMessage("no book with that id")
		responseRecorder := httptest.NewRecorder()
		Convey("When the error response is sent", func() {
			httpx.SendErrorResponse(responseRecorder, err)
			Convey("Then it writes the merry HTTP code and user message as JSON", func() {
				So(responseRecorder.Code, ShouldEqual, http.StatusNotFound)
				var body httpx.ResponseMessage
				decodeErr := json.NewDecoder(responseRecorder.Body).Decode(&body)
				So(decodeErr, ShouldBeNil)
				So(body.Message, ShouldEqual, "no book with that id")
			})
		})
	})
	Convey("Given a plain error with no merry HTTP code or user message", t, func() {
		err := errors.New("boom")
		responseRecorder := httptest.NewRecorder()
		Convey("When the error response is sent", func() {
			httpx.SendErrorResponse(responseRecorder, err)
			Convey("Then it falls back to a 500 with a generic message", func() {
				So(responseRecorder.Code, ShouldEqual, http.StatusInternalServerError)
				var body httpx.ResponseMessage
				decodeErr := json.NewDecoder(responseRecorder.Body).Decode(&body)
				So(decodeErr, ShouldBeNil)
				So(body.Message, ShouldEqual, "internal server error")
			})
		})
	})
}
