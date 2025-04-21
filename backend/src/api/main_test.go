package api_test

import (
	"context"
	"log"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dbutil"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/snapcore/snapd/osutil"
)

var testContext = context.Background()
var testContextCancel context.CancelFunc
var apiContextState dto.ContextState
var exported_share_repo repository.ExportedShareRepositoryInterface

func TestMain(m *testing.M) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	slog.SetLogLoggerLevel(slog.LevelDebug)

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
		log.Fatalf("Cant load config file %#+v", err)
	}
	config.UpdateChannel = string(dto.None)
	mount_repo := repository.NewMountPointPathRepository(dbom.GetDB())
	exported_share_repo = repository.NewExportedShareRepository(dbom.GetDB())
	err = dbutil.FirstTimeJSONImporter(config, mount_repo, exported_share_repo)
	if err != nil {
		log.Fatalf("Cant load json settings - %#+v", err)
	}
	// End

	// Get options
	//options := config.ReadOptionsFile("../../test/data/options.json")
	templateData, err := os.ReadFile("../templates/smb.gtpl")
	if err != nil {
		log.Fatalf("Cant read template file %#+v", err)
	}

	apiContextState = dto.ContextState{}
	apiContextState.SambaConfigFile = "../../test/data/smb.conf"
	apiContextState.Template = templateData
	apiContextState.DockerInterface = "hassio"
	apiContextState.DockerNet = "172.30.32.0/23"
	apiContextState.Heartbeat = 1
	//sharedResources.SSEBroker = NewMockBrokerInterface(ctrl) //
	//testContext = api.StateToContext(&apiContextState, testContext)
	testContext = config.ToContext(testContext)
	testContext, testContextCancel = context.WithCancel(testContext)

	retErr := m.Run()

	testContextCancel()
	testContext.Value("wg").(*sync.WaitGroup).Wait()

	os.Exit(retErr)

}
