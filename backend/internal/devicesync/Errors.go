package devicesync

import (
	"errors"
	"fmt"
)

var (
	ErrDeviceDetailsRequired   = errors.New("device name and host are required")
	ErrDeviceAlreadyRegistered = errors.New("device already registered")
	ErrPasswordRequired        = errors.New("pairing password is required")
)

// DeviceAlreadyRegisteredError is ErrDeviceAlreadyRegistered naming the
// device that already has the host.
type DeviceAlreadyRegisteredError struct {
	Host string
	Name string
}

func (err DeviceAlreadyRegisteredError) Error() string {
	return fmt.Sprintf("%s is already registered as %q", err.Host, err.Name)
}

func (err DeviceAlreadyRegisteredError) Unwrap() error {
	return ErrDeviceAlreadyRegistered
}
