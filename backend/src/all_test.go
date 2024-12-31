package main

import (
	"log"
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/data"
)

func TestMain(m *testing.M) {
	// Get config
	aconfig, err := config.LoadConfig("../test/data/config.json")
	if err != nil {
		log.Fatalf("Cant load config file %s", err)
	}
	data.Config = aconfig

	// Get options
	options = config.ReadOptionsFile("../test/data/options.json")

	// smbConfigFile
	smbConfigFile = new(string)
	*smbConfigFile = "../test/data/smb.conf"

	// Template
	templateDatan, err := defaultTemplate.ReadFile("templates/smb.gtpl")
	if err != nil {
		log.Fatalf("Cant read template file %s", err)
	}
	templateData = templateDatan
	data.ROMode = new(bool)

	config.InitDB(":memory:")

	m.Run()
}
