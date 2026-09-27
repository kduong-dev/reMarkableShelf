package main

import (
	"context"
	"net/http"
	"time"

	"github.com/kduong-dev/goutil/config"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/goutil/logx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/devicesync"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/httpapi"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/openlibrary"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/remarkable"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/store"
)

func main() {
	port := config.EnvString("PORT", "8080")
	dbPath := config.EnvString("DB_PATH", "remarkable-shelf.db")

	db, err := store.Open(dbPath)
	fatal.OnError(err, "opening database: ")
	defer db.Close()

	syncer := devicesync.NewSyncer(db, remarkable.NewSSHClient(remarkable.Config{
		User:     config.EnvString("SSH_USER", "root"),
		Password: config.EnvString("SSH_PASSWORD", ""),
		Port:     config.EnvString("SSH_PORT", "22"),
	}))
	// A zero SYNC_INTERVAL turns background sync off, leaving only Sync now.
	syncInterval := config.EnvDuration("SYNC_INTERVAL", 5*time.Minute)
	if syncInterval > 0 {
		go syncer.Run(context.Background(), syncInterval)
		logx.Noticef("syncing devices in the background every %s", syncInterval)
	}

	handler := httpapi.NewHandler(httpapi.NewHandlerInput{
		Store:       db,
		OpenLibrary: openlibrary.NewClient(),
		Syncer:      syncer,
	})

	logx.Noticef("reMarkable Shelf server listening on :%s (db=%s)", port, dbPath)
	err = http.ListenAndServe(":"+port, handler)
	fatal.OnError(err, "server exited: ")
}
