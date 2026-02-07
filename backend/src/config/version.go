package config

import "fmt"

var (
	Version            = "0.0.0-dev.0"
	CommitHash         = "n/a"
	BuildTimestamp     = "n/a"
	Repository         = "dianlight/srat"
	RollbarToken       = ""
	RollbarEnvironment = ""
	GistToken          = ""
)

func BuildVersion() string {
	return fmt.Sprintf("%s (%s %s)", Version, CommitHash, BuildTimestamp)
}
