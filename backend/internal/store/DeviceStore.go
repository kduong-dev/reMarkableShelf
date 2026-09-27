package store

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/ansel1/merry"
	"github.com/google/uuid"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

func (s *Store) ListDevices() ([]models.Device, error) {
	rows, err := s.DB.Query(`SELECT id, name, host, last_synced_at, paired_at, host_key, identity_changed FROM devices ORDER BY name`)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	defer rows.Close()

	devices := []models.Device{}
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, merry.Wrap(rows.Err())
}

func (s *Store) GetDevice(id string) (models.Device, error) {
	row := s.DB.QueryRow(`SELECT id, name, host, last_synced_at, paired_at, host_key, identity_changed FROM devices WHERE id = ?`, id)
	d, err := scanDevice(row)
	if err == sql.ErrNoRows {
		return models.Device{}, merry.New("device not found").WithHTTPCode(http.StatusNotFound).WithUserMessagef("no device with id %q", id)
	}
	return d, err
}

func (s *Store) CreateDevice(device models.Device) (models.Device, error) {
	if device.ID == "" {
		device.ID = uuid.NewString()
	}
	_, err := s.DB.Exec(`INSERT INTO devices (id, name, host, paired_at, host_key) VALUES (?, ?, ?, ?, ?)`, device.ID, device.Name, device.Host, device.PairedAt, device.HostKey)
	if err != nil {
		return models.Device{}, merry.Wrap(err).WithUserMessage("creating device")
	}
	return device, nil
}

func (s *Store) TouchDeviceSyncedAt(id string, when time.Time) error {
	_, err := s.DB.Exec(`UPDATE devices SET last_synced_at = ? WHERE id = ?`, when, id)
	return merry.Wrap(err)
}

// SetDevicePairedAt records when the device was paired, or with nil that it
// no longer accepts the server's key.
func (s *Store) SetDevicePairedAt(id string, when *time.Time) error {
	_, err := s.DB.Exec(`UPDATE devices SET paired_at = ? WHERE id = ?`, when, id)
	return merry.Wrap(err)
}

// MarkDevicePaired records a successful pairing and the host key the tablet
// presented, clearing any earlier identity change.
func (s *Store) MarkDevicePaired(id string, when time.Time, hostKey string) error {
	_, err := s.DB.Exec(`UPDATE devices SET paired_at = ?, host_key = ?, identity_changed = 0 WHERE id = ?`, when, hostKey, id)
	return merry.Wrap(err)
}

// SetDeviceHostKey records the host key of a device paired before host keys
// were recorded.
func (s *Store) SetDeviceHostKey(id, hostKey string) error {
	_, err := s.DB.Exec(`UPDATE devices SET host_key = ? WHERE id = ?`, hostKey, id)
	return merry.Wrap(err)
}

// MarkDeviceIdentityChanged unpairs a device whose tablet presented a host
// key other than the recorded one, so it isn't synced until re-paired.
func (s *Store) MarkDeviceIdentityChanged(id string) error {
	_, err := s.DB.Exec(`UPDATE devices SET paired_at = NULL, identity_changed = 1 WHERE id = ?`, id)
	return merry.Wrap(err)
}

// DeleteDevice removes a device and, by cascade, its synced documents.
func (s *Store) DeleteDevice(id string) error {
	result, err := s.DB.Exec(`DELETE FROM devices WHERE id = ?`, id)
	if err != nil {
		return merry.Wrap(err).WithUserMessage("removing device")
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return merry.New("device not found").WithHTTPCode(http.StatusNotFound).WithUserMessagef("no device with id %q", id)
	}
	return nil
}

func scanDevice(row rowScanner) (models.Device, error) {
	var d models.Device
	err := row.Scan(&d.ID, &d.Name, &d.Host, &d.LastSyncedAt, &d.PairedAt, &d.HostKey, &d.IdentityChanged)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Device{}, err
		}
		return models.Device{}, merry.Wrap(err)
	}
	return d, nil
}
