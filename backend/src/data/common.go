package data

import nconfig "github.com/dianlight/srat/config"

var Config *nconfig.Config
var ROMode *bool
var UpdateFilePath string
var DirtySectionState nconfig.ConfigSectionDirtySate = nconfig.ConfigSectionDirtySate{
	Shares:   false,
	Users:    false,
	Volumes:  false,
	Settings: false,
}
