package api

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/dianlight/srat/config"
)

var testContext = context.Background()

func TestMain(m *testing.M) {
	// Get config
	aconfig, err := config.LoadConfig("../../test/data/config.json")
	if err != nil {
		log.Fatalf("Cant load config file %s", err)
	}

	// Get options
	options := config.ReadOptionsFile("../../test/data/options.json")

	testContext = context.WithValue(testContext, "addon_config", aconfig)
	testContext = context.WithValue(testContext, "addon_option", options)

	// smbConfigFile
	//smbConfigFile := new(string)
	//*smbConfigFile = "../test/data/smb.conf"

	// Template
	/*
		templateDatan, err := io.ReadFile("../templates/smb.gtpl")
		if err != nil {
			log.Fatalf("Cant read template file %s", err)
		}
		templateData = templateDatan
		data.ROMode = new(bool)
	*/

	//dbom.InitDB(":memory:")

	os.Exit(m.Run())
}
