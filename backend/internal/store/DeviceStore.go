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
	rows, err := s.DB.Query(`SELECT id, name, host, last_synced_at FROM devices ORDER BY name`)
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
	row := s.DB.QueryRow(`SELECT id, name, host, last_synced_at FROM devices WHERE id = ?`, id)
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
	_, err := s.DB.Exec(`INSERT INTO devices (id, name, host) VALUES (?, ?, ?)`, device.ID, device.Name, device.Host)
	if err != nil {
		return models.Device{}, merry.Wrap(err).WithUserMessage("creating device")
	}
	return device, nil
}

func (s *Store) TouchDeviceSyncedAt(id string, when time.Time) error {
	_, err := s.DB.Exec(`UPDATE devices SET last_synced_at = ? WHERE id = ?`, when, id)
	return merry.Wrap(err)
}

func scanDevice(row rowScanner) (models.Device, error) {
	var d models.Device
	err := row.Scan(&d.ID, &d.Name, &d.Host, &d.LastSyncedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Device{}, err
		}
		return models.Device{}, merry.Wrap(err)
	}
	return d, nil
}
