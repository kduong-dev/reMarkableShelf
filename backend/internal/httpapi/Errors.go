package httpapi

import (
	"net/http"

	"github.com/ansel1/merry"
)

var ErrInvalidYear = merry.New("invalid year").WithHTTPCode(http.StatusBadRequest)
