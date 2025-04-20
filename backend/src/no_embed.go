//go:build !embedallowed
// +build !embedallowed

package main

import (
	"log"
	"net/http"
	"os"
)

var is_embed = false

func getFrontend() (fs http.FileSystem) {

	if frontend == nil || *frontend == "" {
		frontend = new(string)
		*frontend = "static"
	}

	dir := http.Dir(*frontend)
	_, err := dir.Open(".")
	if err != nil {
		log.Fatalf("Cant access frontend folder %s - %s", *frontend, err)
	}
	return dir
}

func getTemplateData() []byte {
	if templateFile == nil || *templateFile == "" {
		templateFile = new(string)
		*templateFile = "templates/smb.gtpl"
	}
	templateDatan, err := os.ReadFile(*templateFile)
	if err != nil {
		log.Fatalf("Cant read template file %s - %s", *templateFile, err)
	}
	return templateDatan
}
