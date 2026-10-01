package sourcestore

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/database"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

const sourceColumns = `id, kind, name, url, search_url, priority, enabled, created_at`

type SQLiteStore struct {
	database *sql.DB
}

func NewSQLiteStore(database *sql.DB) *SQLiteStore {
	return &SQLiteStore{database: database}
}

func (store *SQLiteStore) List() ([]models.EbookSource, error) {
	rows, err := store.database.Query(`SELECT ` + sourceColumns + ` FROM ebook_sources ORDER BY priority, created_at`)
	if err != nil {
		return nil, fmt.Errorf("listing ebook sources: %w", err)
	}
	defer rows.Close()
	sources := []models.EbookSource{}
	for rows.Next() {
		source, err := scanSource(rows)
		if err != nil {
			return nil, fmt.Errorf("listing ebook sources: %w", err)
		}
		sources = append(sources, source)
	}
	return sources, rows.Err()
}

func (store *SQLiteStore) Get(id string) (models.EbookSource, error) {
	source, err := scanSource(store.database.QueryRow(`SELECT `+sourceColumns+` FROM ebook_sources WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return models.EbookSource{}, fmt.Errorf("ebook source %q: %w", id, ErrSourceNotFound)
	}
	if err != nil {
		return models.EbookSource{}, fmt.Errorf("getting ebook source: %w", err)
	}
	return source, nil
}

func (store *SQLiteStore) Create(source models.EbookSource) (models.EbookSource, error) {
	if source.Name == "" {
		return models.EbookSource{}, ErrSourceNameNeeded
	}
	var existing int
	err := store.database.QueryRow(`SELECT COUNT(*) FROM ebook_sources WHERE kind = ? AND url = ?`, source.Kind, source.URL).Scan(&existing)
	if err != nil {
		return models.EbookSource{}, fmt.Errorf("adding ebook source: %w", err)
	}
	if existing > 0 {
		return models.EbookSource{}, fmt.Errorf("%w: %s", ErrSourceExists, source.URL)
	}
	source.ID = uuid.NewString()
	source.Enabled = true
	source.CreatedAt = time.Now().UTC()
	err = store.database.QueryRow(`SELECT COALESCE(MAX(priority), -1) + 1 FROM ebook_sources`).Scan(&source.Priority)
	if err != nil {
		return models.EbookSource{}, fmt.Errorf("adding ebook source: %w", err)
	}
	_, err = store.database.Exec(`INSERT INTO ebook_sources (`+sourceColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		source.ID, source.Kind, source.Name, source.URL, source.SearchURL, source.Priority, source.Enabled, source.CreatedAt)
	if err != nil {
		return models.EbookSource{}, fmt.Errorf("adding ebook source: %w", err)
	}
	return source, nil
}

func (store *SQLiteStore) SetEnabled(id string, enabled bool) (models.EbookSource, error) {
	result, err := store.database.Exec(`UPDATE ebook_sources SET enabled = ? WHERE id = ?`, enabled, id)
	if err != nil {
		return models.EbookSource{}, fmt.Errorf("updating ebook source: %w", err)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return models.EbookSource{}, fmt.Errorf("ebook source %q: %w", id, ErrSourceNotFound)
	}
	return store.Get(id)
}

func (store *SQLiteStore) Reorder(ids []string) error {
	sources, err := store.List()
	if err != nil {
		return err
	}
	current := make([]string, 0, len(sources))
	for _, source := range sources {
		current = append(current, source.ID)
	}
	if !sameIDs(current, ids) {
		return ErrInvalidOrder
	}
	tx, err := store.database.Begin()
	if err != nil {
		return fmt.Errorf("reordering ebook sources: %w", err)
	}
	defer tx.Rollback()
	for priority, id := range ids {
		if _, err := tx.Exec(`UPDATE ebook_sources SET priority = ? WHERE id = ?`, priority, id); err != nil {
			return fmt.Errorf("reordering ebook sources: %w", err)
		}
	}
	return tx.Commit()
}

func (store *SQLiteStore) Delete(id string) error {
	if id == models.BuiltInEbookSourceID {
		return ErrBuiltInSource
	}
	result, err := store.database.Exec(`DELETE FROM ebook_sources WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("removing ebook source: %w", err)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return fmt.Errorf("ebook source %q: %w", id, ErrSourceNotFound)
	}
	return nil
}

// sameIDs reports whether ids lists exactly the current ids, each once.
func sameIDs(current, ids []string) bool {
	sortedCurrent, sortedIDs := slices.Clone(current), slices.Clone(ids)
	slices.Sort(sortedCurrent)
	slices.Sort(sortedIDs)
	return slices.Equal(sortedCurrent, sortedIDs)
}

func scanSource(row database.RowScanner) (models.EbookSource, error) {
	var source models.EbookSource
	err := row.Scan(&source.ID, &source.Kind, &source.Name, &source.URL, &source.SearchURL, &source.Priority, &source.Enabled, &source.CreatedAt)
	return source, err
}
