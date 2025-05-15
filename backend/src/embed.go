//go:build embedallowed

package main

//go:generate make -C .. static

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

var is_embed = true

//go:embed static/*
var static_content embed.FS

func getFrontend() http.FileSystem {
	fsRoot, _ := fs.Sub(static_content, "static")

	return http.FS(fsRoot)
}

//go:embed templates/smb.gtpl
var template_content embed.FS

func getTemplateData() []byte {
	templateDatan, err := template_content.ReadFile("templates/smb.gtpl")
	if err != nil {
		log.Fatalf("Cant read template file %s - %s", "templates/smb.gtpl", err)
	}
	return templateDatan
}
