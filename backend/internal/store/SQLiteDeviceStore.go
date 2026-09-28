package store

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/ansel1/merry"
	"github.com/google/uuid"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

type SQLiteDeviceStore struct {
	database *sql.DB
}

func (deviceStore *SQLiteDeviceStore) List() ([]models.Device, error) {
	rows, err := deviceStore.database.Query(`SELECT id, name, host, last_synced_at, paired_at, host_key, identity_changed FROM devices ORDER BY name`)
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

func (deviceStore *SQLiteDeviceStore) Get(id string) (models.Device, error) {
	row := deviceStore.database.QueryRow(`SELECT id, name, host, last_synced_at, paired_at, host_key, identity_changed FROM devices WHERE id = ?`, id)
	d, err := scanDevice(row)
	if err == sql.ErrNoRows {
		return models.Device{}, merry.New("device not found").WithHTTPCode(http.StatusNotFound).WithUserMessagef("no device with id %q", id)
	}
	return d, err
}

func (deviceStore *SQLiteDeviceStore) Create(device models.Device) (models.Device, error) {
	if device.ID == "" {
		device.ID = uuid.NewString()
	}
	_, err := deviceStore.database.Exec(`INSERT INTO devices (id, name, host, paired_at, host_key) VALUES (?, ?, ?, ?, ?)`, device.ID, device.Name, device.Host, device.PairedAt, device.HostKey)
	if err != nil {
		return models.Device{}, merry.Wrap(err).WithUserMessage("creating device")
	}
	return device, nil
}

func (deviceStore *SQLiteDeviceStore) TouchSyncedAt(id string, when time.Time) error {
	_, err := deviceStore.database.Exec(`UPDATE devices SET last_synced_at = ? WHERE id = ?`, when, id)
	return merry.Wrap(err)
}

func (deviceStore *SQLiteDeviceStore) SetPairedAt(id string, when *time.Time) error {
	_, err := deviceStore.database.Exec(`UPDATE devices SET paired_at = ? WHERE id = ?`, when, id)
	return merry.Wrap(err)
}

func (deviceStore *SQLiteDeviceStore) MarkPaired(id string, when time.Time, hostKey string) error {
	_, err := deviceStore.database.Exec(`UPDATE devices SET paired_at = ?, host_key = ?, identity_changed = 0 WHERE id = ?`, when, hostKey, id)
	return merry.Wrap(err)
}

func (deviceStore *SQLiteDeviceStore) SetHostKey(id, hostKey string) error {
	_, err := deviceStore.database.Exec(`UPDATE devices SET host_key = ? WHERE id = ?`, hostKey, id)
	return merry.Wrap(err)
}

func (deviceStore *SQLiteDeviceStore) MarkIdentityChanged(id string) error {
	_, err := deviceStore.database.Exec(`UPDATE devices SET paired_at = NULL, identity_changed = 1 WHERE id = ?`, id)
	return merry.Wrap(err)
}

func (deviceStore *SQLiteDeviceStore) Rename(id, name string) (models.Device, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return models.Device{}, merry.Here(ErrDeviceNameRequired)
	}
	result, err := deviceStore.database.Exec(`UPDATE devices SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return models.Device{}, merry.Wrap(err).WithUserMessage("renaming device")
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return models.Device{}, merry.New("device not found").WithHTTPCode(http.StatusNotFound).WithUserMessagef("no device with id %q", id)
	}
	return deviceStore.Get(id)
}

func (deviceStore *SQLiteDeviceStore) Delete(id string) error {
	result, err := deviceStore.database.Exec(`DELETE FROM devices WHERE id = ?`, id)
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
