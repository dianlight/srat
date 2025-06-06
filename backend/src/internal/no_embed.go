//go:build !embedallowed

package internal

import (
	"log/slog"
	"net/http"
	"os"

	"gitlab.com/tozd/go/errors"
)

var TemplateFile *string
var Frontend *string

var Is_embed = false

func GetFrontend() (fs http.FileSystem) {

	if Frontend == nil || *Frontend == "" {
		Frontend = new(string)
		*Frontend = "web/static"
	}

	dir := http.Dir(*Frontend)
	_, err := dir.Open(".")
	if err != nil {
		slog.Error("Cant access frontend folder", "Folder:", *Frontend, "Err", errors.WithStack(err))
	}
	return dir
}

func GetTemplateData() []byte {
	if TemplateFile == nil || *TemplateFile == "" {
		TemplateFile = new(string)
		*TemplateFile = "templates/smb.gtpl"
	}
	templateDatan, err := os.ReadFile(*TemplateFile)
	if err != nil {
		slog.Error("Cant read template file", "File:", *TemplateFile, "Err", errors.WithStack(err))
	}
	return templateDatan
}
