package service_test

import (
	"context"
	"log"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dbutil"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/lmittmann/tint"
	"github.com/snapcore/snapd/osutil"
)

var testContext, testContextCancel = context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
var apiContextState dto.ContextState
var exported_share_repo repository.ExportedShareRepositoryInterface

func TestMain(m *testing.M) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.RFC3339,
			//NoColor:    !isatty.IsTerminal(os.Stderr.Fd()),
			AddSource: true,
		}),
	))
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

	retErr := m.Run()
	testContextCancel()
	testContext.Value("wg").(*sync.WaitGroup).Wait()

	os.Exit(retErr)

}
