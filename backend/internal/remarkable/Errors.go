package remarkable

import (
	"net/http"

	"github.com/ansel1/merry"
)

var (
	ErrAuthenticationFailed = merry.New("tablet rejected ssh credentials").WithHTTPCode(http.StatusBadGateway)
	ErrTabletUnreachable    = merry.New("tablet unreachable").WithHTTPCode(http.StatusGatewayTimeout)
)
