package store

import (
	"net/http"

	"github.com/ansel1/merry"
)

var (
	ErrInvalidPageCount      = merry.New("page count must be at least 1").WithHTTPCode(http.StatusBadRequest).WithUserMessage("the page count must be at least 1")
	ErrDeviceNameRequired    = merry.New("device name is required").WithHTTPCode(http.StatusBadRequest).WithUserMessage("enter a name for the device")
	ErrCurrentPageOutOfRange = merry.New("current page is out of range").WithHTTPCode(http.StatusBadRequest).WithUserMessage("the page can't be negative or past the end of the book")
)
