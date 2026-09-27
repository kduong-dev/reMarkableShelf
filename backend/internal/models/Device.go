package models

import "time"

// Device is a registered reMarkable tablet, identified by its SSH host.
type Device struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Host         string     `json:"host"`
	LastSyncedAt *time.Time `json:"lastSyncedAt,omitempty"`
	PairedAt     *time.Time `json:"pairedAt,omitempty"`
}
