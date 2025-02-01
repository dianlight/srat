package service_test

import (
	"context"
	"log"
	"log/slog"
	"os"
	"testing"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dbutil"
	"github.com/dianlight/srat/dto"
	"github.com/snapcore/snapd/osutil"
	"github.com/ztrue/tracerr"
)

var testContext = context.Background()
var apiContextState api.ContextState

func TestMain(m *testing.M) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	slog.SetLogLoggerLevel(slog.LevelDebug)

	os.Setenv("HOSTNAME", "test-host")

	data, err := os.ReadFile("../../test/data/mount_info.txt")
	if err != nil {
		log.Fatal(err)
	}
	osutil.MockMountInfo(string(data))

	dbom.InitDB("file::memory:?cache=shared&_pragma=foreign_keys(1)")
	defer dbom.CloseDB()

	var config config.Config
	err = config.LoadConfig("../../test/data/config.json")
	// Setting/Properties
	if err != nil {
		log.Fatalf("Cant load config file %s", err)
	}
	config.UpdateChannel = string(dto.None)
	err = dbutil.FirstTimeJSONImporter(config)
	if err != nil {
		log.Fatalf("Cant load json settings - %v", tracerr.SprintSourceColor(err))
	}
	// End

	retErr := m.Run()

	os.Exit(retErr)

}
