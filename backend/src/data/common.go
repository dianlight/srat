package data

import (
	nconfig "github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dm"
)

var Config *nconfig.Config
var ROMode *bool
var UpdateFilePath string
var DirtySectionState dm.DataDirtyTracker = dm.DataDirtyTracker{
	Shares:   false,
	Users:    false,
	Volumes:  false,
	Settings: false,
}
var ConfigFile *string
var SupervisorToken *string
