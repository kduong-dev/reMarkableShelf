package main

import (
	"net/http"

	"github.com/kduong-dev/goutil/config"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/goutil/logx"
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

	handler := httpapi.NewHandler(httpapi.NewHandlerInput{
		Store:       db,
		OpenLibrary: openlibrary.NewClient(),
		RemarkableConfig: remarkable.Config{
			User:     config.EnvString("SSH_USER", "root"),
			Password: config.EnvString("SSH_PASSWORD", ""),
			Port:     config.EnvString("SSH_PORT", "22"),
		},
	})

	logx.Noticef("reMarkable Shelf server listening on :%s (db=%s)", port, dbPath)
	err = http.ListenAndServe(":"+port, handler)
	fatal.OnError(err, "server exited: ")
}
