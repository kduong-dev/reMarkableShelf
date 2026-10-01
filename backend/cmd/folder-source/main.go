// folder-source is an example ebook source plugin: it serves the EPUBs and
// PDFs under FOLDER_SOURCE_DIR, named FOLDER_SOURCE_NAME on the Sources
// page, on PORT.
package main

import (
	"net/http"

	"github.com/kduong-dev/goutil/config"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/goutil/logx"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/foldersource"
	"github.com/kduong-dev/reMarkableShelf/backend/pkg/sourceplugin"
)

func main() {
	port := config.EnvString("PORT", "8090")
	directory := config.EnvStringOrFatal("FOLDER_SOURCE_DIR")
	name := config.EnvString("FOLDER_SOURCE_NAME", "Ebook folder")
	logx.Noticef("folder-source serving %s as %q on :%s", directory, name, port)
	err := http.ListenAndServe(":"+port, sourceplugin.NewHandler(foldersource.New(name, directory)))
	fatal.OnError(err, "server exited: ")
}
