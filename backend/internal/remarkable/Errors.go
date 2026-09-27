package remarkable

import (
	"net/http"

	"github.com/ansel1/merry"
)

var (
	ErrWrongPassword     = merry.New("tablet rejected the pairing password").WithHTTPCode(http.StatusBadRequest).WithUserMessage("the tablet rejected that password")
	ErrNotPaired         = merry.New("tablet rejected the server's key").WithHTTPCode(http.StatusBadGateway).WithUserMessage("the tablet no longer accepts this server's key, so pair it again")
	ErrKeyNotAccepted    = merry.New("tablet rejected the key just installed").WithHTTPCode(http.StatusBadGateway).WithUserMessage("the key was installed but the tablet still won't accept it")
	ErrHostKeyChanged    = merry.New("tablet presented a different host key than when paired").WithHTTPCode(http.StatusBadGateway).WithUserMessage("the tablet identified itself differently than when it was paired; if you factory-reset or replaced it, pair it again, otherwise something on your network may be impersonating it")
	ErrTabletUnreachable = merry.New("tablet unreachable").WithHTTPCode(http.StatusGatewayTimeout).WithUserMessage("couldn't reach the tablet; it may be asleep or off the network")
)
