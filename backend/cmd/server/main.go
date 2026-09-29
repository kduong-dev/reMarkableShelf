package main

import (
	"context"
	"net/http"
	"path/filepath"
	"time"

	"github.com/kduong-dev/goutil/config"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/goutil/logx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookfilestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/database"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicestore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicesync"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/documentstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/httpapi"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/internetarchive"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/remarkable"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

func main() {
	port := config.EnvString("PORT", "8080")
	dbPath := config.EnvString("DB_PATH", "remarkable-shelf.db")

	opened, err := database.Open(dbPath)
	fatal.OnError(err, "opening database: ")
	defer opened.Close()
	bookStore := bookstore.NewSQLiteStore(opened)
	deviceStore := devicestore.NewSQLiteStore(opened)
	documentStore := documentstore.NewSQLiteStore(opened)
	// Book files live in the storage service, reached with
	// STORAGE_SERVICE_CLIENT_IMPLEMENTATION=HTTP, STORAGE_SERVICE_URL and
	// STORAGE_SERVICE_API_KEY.
	bookFileStore := bookfilestore.NewStorageServiceStore(opened, storageservice.ClientFromEnv())

	// The server's key signs in to every paired tablet, so it lives beside
	// the database on persistent storage unless SSH_KEY_PATH says otherwise.
	keyPath := config.EnvString("SSH_KEY_PATH", filepath.Join(filepath.Dir(dbPath), "remarkable-shelf_ed25519"))
	signer, err := remarkable.LoadOrCreateSigner(keyPath)
	fatal.OnError(err, "loading SSH key: ")
	syncer := devicesync.NewSyncer(devicesync.NewSyncerInput{
		BookFileStore: bookFileStore,
		BookStore:     bookStore,
		DeviceStore:   deviceStore,
		DocumentStore: documentStore,
		Remarkable: remarkable.NewSSHClient(remarkable.Config{
			User:   config.EnvString("SSH_USER", "root"),
			Port:   config.EnvString("SSH_PORT", "22"),
			Signer: signer,
		}),
	})
	// A zero SYNC_INTERVAL turns background sync off, leaving only Sync now.
	syncInterval := config.EnvDuration("SYNC_INTERVAL", 5*time.Minute)
	if syncInterval > 0 {
		go syncer.Run(context.Background(), syncInterval)
		logx.Noticef("syncing devices in the background every %s", syncInterval)
	}

	handler := httpapi.NewHandler(httpapi.NewHandlerInput{
		BookFileStore:   bookFileStore,
		BookStore:       bookStore,
		DeviceStore:     deviceStore,
		DocumentStore:   documentStore,
		InternetArchive: internetarchive.NewClient(),
		OpenLibrary:     openlibrary.NewClient(),
		Syncer:          syncer,
	})

	logx.Noticef("reMarkable Shelf server listening on :%s (db=%s)", port, dbPath)
	err = http.ListenAndServe(":"+port, handler)
	fatal.OnError(err, "server exited: ")
}
