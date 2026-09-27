package httpapi

import (
	"net/http"

	"github.com/ansel1/merry"
)

var ErrInvalidNumber = merry.New("invalid number").WithHTTPCode(http.StatusBadRequest)
