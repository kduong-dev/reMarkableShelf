package remarkable

import "errors"

var (
	ErrWrongPassword     = errors.New("tablet rejected the pairing password")
	ErrNotPaired         = errors.New("tablet rejected the server's key")
	ErrKeyNotAccepted    = errors.New("tablet rejected the key just installed")
	ErrHostKeyChanged    = errors.New("tablet presented a different host key than when paired")
	ErrTabletUnreachable = errors.New("tablet unreachable")
	ErrInvalidDocument   = errors.New("invalid document to copy")
	ErrRestartFailed     = errors.New("tablet's reading app didn't restart")
)
