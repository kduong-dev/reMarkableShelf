package models

import "time"

// Device is a registered reMarkable tablet, identified by its SSH host.
type Device struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Host         string     `json:"host"`
	LastSyncedAt *time.Time `json:"lastSyncedAt,omitempty"`
	PairedAt     *time.Time `json:"pairedAt,omitempty"`
	// HostKey is the SSH host key the tablet presented when paired; the
	// server refuses to connect if it ever presents another, and sets
	// IdentityChanged.
	HostKey         string `json:"-"`
	IdentityChanged bool   `json:"identityChanged,omitempty"`
	// DownloadsFolderUUID is the tablet folder books saved on the server are
	// copied into, "" until sync first creates or finds it.
	DownloadsFolderUUID string `json:"downloadsFolderUuid,omitempty"`
}
