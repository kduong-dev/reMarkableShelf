package devicestore

import "errors"

var (
	ErrDeviceNotFound     = errors.New("device not found")
	ErrDeviceNameRequired = errors.New("device name is required")
)
