package devicestore

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/database"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

type SQLiteStore struct {
	database *sql.DB
}

func NewSQLiteStore(database *sql.DB) *SQLiteStore {
	return &SQLiteStore{database: database}
}

func (deviceStore *SQLiteStore) List() ([]models.Device, error) {
	rows, err := deviceStore.database.Query(`SELECT id, name, host, last_synced_at, paired_at, host_key, identity_changed FROM devices ORDER BY name`)
	if err != nil {
		return nil, err
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
	return devices, rows.Err()
}

func (deviceStore *SQLiteStore) Get(id string) (models.Device, error) {
	row := deviceStore.database.QueryRow(`SELECT id, name, host, last_synced_at, paired_at, host_key, identity_changed FROM devices WHERE id = ?`, id)
	d, err := scanDevice(row)
	if err == sql.ErrNoRows {
		return models.Device{}, fmt.Errorf("device %q: %w", id, ErrDeviceNotFound)
	}
	return d, err
}

func (deviceStore *SQLiteStore) Create(device models.Device) (models.Device, error) {
	if device.ID == "" {
		device.ID = uuid.NewString()
	}
	_, err := deviceStore.database.Exec(`INSERT INTO devices (id, name, host, paired_at, host_key) VALUES (?, ?, ?, ?, ?)`, device.ID, device.Name, device.Host, device.PairedAt, device.HostKey)
	if err != nil {
		return models.Device{}, fmt.Errorf("creating device: %w", err)
	}
	return device, nil
}

func (deviceStore *SQLiteStore) TouchSyncedAt(id string, when time.Time) error {
	_, err := deviceStore.database.Exec(`UPDATE devices SET last_synced_at = ? WHERE id = ?`, when, id)
	return err
}

func (deviceStore *SQLiteStore) SetPairedAt(id string, when *time.Time) error {
	_, err := deviceStore.database.Exec(`UPDATE devices SET paired_at = ? WHERE id = ?`, when, id)
	return err
}

func (deviceStore *SQLiteStore) MarkPaired(id string, when time.Time, hostKey string) error {
	_, err := deviceStore.database.Exec(`UPDATE devices SET paired_at = ?, host_key = ?, identity_changed = 0 WHERE id = ?`, when, hostKey, id)
	return err
}

func (deviceStore *SQLiteStore) SetHostKey(id, hostKey string) error {
	_, err := deviceStore.database.Exec(`UPDATE devices SET host_key = ? WHERE id = ?`, hostKey, id)
	return err
}

func (deviceStore *SQLiteStore) MarkIdentityChanged(id string) error {
	_, err := deviceStore.database.Exec(`UPDATE devices SET paired_at = NULL, identity_changed = 1 WHERE id = ?`, id)
	return err
}

func (deviceStore *SQLiteStore) Rename(id, name string) (models.Device, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return models.Device{}, ErrDeviceNameRequired
	}
	result, err := deviceStore.database.Exec(`UPDATE devices SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return models.Device{}, fmt.Errorf("renaming device: %w", err)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return models.Device{}, fmt.Errorf("device %q: %w", id, ErrDeviceNotFound)
	}
	return deviceStore.Get(id)
}

func (deviceStore *SQLiteStore) Delete(id string) error {
	result, err := deviceStore.database.Exec(`DELETE FROM devices WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("removing device: %w", err)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return fmt.Errorf("device %q: %w", id, ErrDeviceNotFound)
	}
	return nil
}

func scanDevice(row database.RowScanner) (models.Device, error) {
	var d models.Device
	err := row.Scan(&d.ID, &d.Name, &d.Host, &d.LastSyncedAt, &d.PairedAt, &d.HostKey, &d.IdentityChanged)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Device{}, err
		}
		return models.Device{}, err
	}
	return d, nil
}
