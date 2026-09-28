package devicestore

import (
	"net/http"

	"github.com/ansel1/merry"
)

var (
	ErrDeviceNameRequired = merry.New("device name is required").WithHTTPCode(http.StatusBadRequest).WithUserMessage("enter a name for the device")
)
