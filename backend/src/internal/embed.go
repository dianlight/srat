//go:build embedallowed

package internal

import (
	"io/fs"
	"log"
	"net/http"

	"github.com/dianlight/srat/templates"
	"github.com/dianlight/srat/web"
)

var TemplateFile *string
var Frontend *string

var Is_embed = true

func GetFrontend() http.FileSystem {
	fsRoot, _ := fs.Sub(web.Static_content, "static")

	return http.FS(fsRoot)
}

func GetTemplateData() []byte {
	templateDatan, err := templates.Template_content.ReadFile("smb.gtpl")
	if err != nil {
		log.Fatalf("Cant read template file %s - %s", "smb.gtpl", err)
	}
	return templateDatan
}
