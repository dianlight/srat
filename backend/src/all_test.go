package main

import (
	"log"
	"testing"
)

func TestMain(m *testing.M) {
	// Get config
	config = readConfig("../test/data/config.json")
	// Get options
	options = readOptionsFile("../test/data/options.json")

	// smbConfigFile
	smbConfigFile = new(string)
	*smbConfigFile = "../test/data/smb.conf"

	// Template
	templateDatan, err := defaultTemplate.ReadFile("templates/smb.gtpl")
	if err != nil {
		log.Fatalf("Cant read template file %s", err)
	}
	templateData = templateDatan

	m.Run()
}
