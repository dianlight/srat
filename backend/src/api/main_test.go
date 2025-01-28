package api

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dbutil"
	"github.com/snapcore/snapd/osutil"
	"github.com/ztrue/tracerr"
)

var testContext = context.Background()

func TestMain(m *testing.M) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	data, err := os.ReadFile("../../test/data/mount_info.txt")
	if err != nil {
		log.Fatal(err)
	}
	osutil.MockMountInfo(string(data))

	os.Setenv("HOSTNAME", "test-host")

	dbom.InitDB("file::memory:?cache=shared&_pragma=foreign_keys(1)")
	defer dbom.CloseDB()

	var config config.Config
	err = config.LoadConfig("../../test/data/config.json")
	// Setting/Properties
	if err != nil {
		log.Fatalf("Cant load config file %s", err)
	}
	err = dbutil.FirstTimeJSONImporter(config)
	if err != nil {
		log.Fatalf("Cant load json settings - %v", tracerr.SprintSourceColor(err))
	}
	// End

	// Get options
	//options := config.ReadOptionsFile("../../test/data/options.json")
	templateData, err := os.ReadFile("../templates/smb.gtpl")
	if err != nil {
		log.Fatalf("Cant read template file %s", err)
	}

	sharedResources := ContextState{}
	sharedResources.SambaConfigFile = "../../test/data/smb.conf"
	sharedResources.Template = templateData
	sharedResources.DockerInterface = "hassio"
	sharedResources.DockerNet = "172.30.32.0/23"
	StateToContext(&sharedResources, testContext)
	testContext = config.ToContext(testContext)

	os.Exit(m.Run())
}
