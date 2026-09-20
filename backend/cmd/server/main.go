package main

import (
	"log"
	"net/http"

	"github.com/kqvd/reMarkableShelf/backend/internal/api"
	"github.com/kqvd/reMarkableShelf/backend/internal/config"
	"github.com/kqvd/reMarkableShelf/backend/internal/fatal"
	"github.com/kqvd/reMarkableShelf/backend/internal/googlebooks"
	"github.com/kqvd/reMarkableShelf/backend/internal/remarkable"
	"github.com/kqvd/reMarkableShelf/backend/internal/store"
)

func main() {
	port := config.EnvString("PORT", "8080")
	dbPath := config.EnvString("DB_PATH", "remarkable-shelf.db")

	db, err := store.Open(dbPath)
	fatal.OnError(err, "opening database: ")
	defer db.Close()

	handler := &api.Handler{
		Store:       db,
		GoogleBooks: googlebooks.NewClient(config.EnvString("GOOGLE_BOOKS_API_KEY", "")),
		RemarkableCfg: remarkable.Config{
			User:     config.EnvString("SSH_USER", "root"),
			Password: config.EnvString("SSH_PASSWORD", ""),
			Port:     config.EnvString("SSH_PORT", "22"),
		},
	}

	log.Printf("reMarkable Shelf server listening on :%s (db=%s)", port, dbPath)
	err = http.ListenAndServe(":"+port, api.NewHandler(handler))
	fatal.OnError(err, "server exited: ")
}
