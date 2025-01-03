package api

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
)

var testContext = context.Background()

func TestMain(m *testing.M) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Get config
	aconfig, err := config.LoadConfig("../../test/data/config.json")
	if err != nil {
		log.Fatalf("Cant load config file %s", err)
	}

	// Get options
	options := config.ReadOptionsFile("../../test/data/options.json")
	templateData, err := os.ReadFile("../templates/smb.gtpl")
	if err != nil {
		log.Fatalf("Cant read template file %s", err)
	}

	testContext = context.WithValue(testContext, "addon_config", aconfig)
	testContext = context.WithValue(testContext, "addon_option", options)
	testContext = context.WithValue(testContext, "data_dirty_tracker", &dto.DataDirtyTracker{})
	var smbConfigFile = "../../test/data/smb.conf"
	testContext = context.WithValue(testContext, "samba_config_file", &smbConfigFile)
	testContext = context.WithValue(testContext, "template_data", templateData)

	// Template
	/*
		templateDatan, err := io.ReadFile("../templates/smb.gtpl")
		if err != nil {
			log.Fatalf("Cant read template file %s", err)
		}
		templateData = templateDatan
		data.ROMode = new(bool)
	*/

	dbom.InitDB(":memory:")

	os.Exit(m.Run())
}
