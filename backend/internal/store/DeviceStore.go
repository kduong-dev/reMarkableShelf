package store

import (
	"time"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

// DeviceStore holds the tablets the server syncs, and whether each one is
// still paired.
type DeviceStore interface {
	List() ([]models.Device, error)
	Get(id string) (models.Device, error)
	Create(device models.Device) (models.Device, error)
	TouchSyncedAt(id string, when time.Time) error
	// SetPairedAt records when the device was paired, or with nil that it
	// no longer accepts the server's key.
	SetPairedAt(id string, when *time.Time) error
	// MarkPaired records a successful pairing and the host key the tablet
	// presented, clearing any earlier identity change.
	MarkPaired(id string, when time.Time, hostKey string) error
	// SetHostKey records the host key of a device paired before host keys
	// were recorded.
	SetHostKey(id, hostKey string) error
	// MarkIdentityChanged unpairs a device whose tablet presented a host key
	// other than the recorded one, so it isn't synced until re-paired.
	MarkIdentityChanged(id string) error
	// Rename changes a device's name, returning the renamed device.
	Rename(id, name string) (models.Device, error)
	// Delete removes a device and its synced documents.
	Delete(id string) error
}
