package devicesync

import (
	"net/http"

	"github.com/ansel1/merry"
)

var (
	ErrDeviceDetailsRequired   = merry.New("device name and host are required").WithHTTPCode(http.StatusBadRequest).WithUserMessage("enter the device's name and host")
	ErrDeviceAlreadyRegistered = merry.New("device already registered").WithHTTPCode(http.StatusConflict)
	ErrPasswordRequired        = merry.New("pairing password is required").WithHTTPCode(http.StatusBadRequest).WithUserMessage("enter the tablet's password to pair it")
)
