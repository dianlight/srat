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

	dbom.InitDB(":memory:")
	defer dbom.CloseDB()

	var config config.Config
	err := config.LoadConfig("../../test/data/config.json")
	// Setting/Properties
	if err != nil {
		log.Fatalf("Cant load config file %s", err)
	}
	var settings dto.Settings
	err = settings.From(&config)
	if err != nil {
		log.Fatalf("Cant save settings - %s", err)
	}
	var properties dbom.Properties
	err = settings.ToArray(&properties)
	if err != nil {
		log.Fatalf("Cant save settings - %s", err)
	}
	err = properties.Save()
	if err != nil {
		log.Fatalf("Cant save properties - %s", err)
	}
	// Users
	var users dto.Users
	err = users.From(&config.OtherUsers)
	if err != nil {
		log.Fatalf("Cant save users - %s", err)
	}
	var sambaUsers dbom.SambaUsers
	err = users.To(&sambaUsers)
	if err != nil {
		log.Fatalf("Cant save users - %s", err)
	}
	err = sambaUsers.Save()
	if err != nil {
		log.Fatalf("Cant save users - %s", err)
	}
	// Shares
	var shares dto.SharedResources
	err = shares.From(&config.Shares)
	if err != nil {
		log.Fatalf("Cant save shares - %s", err)
	}
	var sambaShares dbom.ExportedShares
	err = shares.ToArray(&sambaShares)
	if err != nil {
		log.Fatalf("Cant save shares - %s", err)
	}
	err = sambaShares.Save()
	if err != nil {
		log.Fatalf("Cant save shares - %s", err)
	}
	// End

	// Get options
	//options := config.ReadOptionsFile("../../test/data/options.json")
	templateData, err := os.ReadFile("../templates/smb.gtpl")
	if err != nil {
		log.Fatalf("Cant read template file %s", err)
	}

	sharedResources := dto.ContextState{}
	//sharedResources.FromJSONConfig(*aconfig)
	testContext = sharedResources.ToContext(testContext)
	//sharedResources := dto.SharedResources{}
	//sharedResources.From(aconfig.Shares)
	//testContext = context.WithValue(testContext, "shared_resources", sharedResources)
	//testContext = context.WithValue(testContext, "addon_option", options)
	//testContext = context.WithValue(testContext, "data_dirty_tracker", &dto.DataDirtyTracker{})
	var smbConfigFile = "../../test/data/smb.conf"
	testContext = config.ToContext(testContext)
	testContext = context.WithValue(testContext, "samba_config_file", &smbConfigFile)
	testContext = context.WithValue(testContext, "template_data", templateData)
	var dockerInterface = "hassio"
	var dockerNetwork = "172.30.32.0/23"
	testContext = context.WithValue(testContext, "docker_interface", &dockerInterface)
	testContext = context.WithValue(testContext, "docker_network", &dockerNetwork)

	//pretty.Print(testContext.Value("context_state"))
	// Template
	/*
		templateDatan, err := io.ReadFile("../templates/smb.gtpl")
		if err != nil {
			log.Fatalf("Cant read template file %s", err)
		}
		templateData = templateDatan
		data.ROMode = new(bool)
	*/

	os.Exit(m.Run())
}
