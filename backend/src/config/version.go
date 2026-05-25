package config

import (
	"fmt"
	"strings"
)

var (
	Version        = "0.0.0-dev.0"
	CommitHash     = "n/a"
	BuildTimestamp = "n/a"
	Repository     = "dianlight/srat"
	SentryDSN      = ""
	GistToken      = ""
)

func BuildVersion() string {
	return fmt.Sprintf("%s (%s %s)", Version, CommitHash, BuildTimestamp)
}

// Environment returns the runtime environment classification derived from the
// current build version.
func Environment() string {
	return EnvironmentFromVersion(Version)
}

// EnvironmentFromVersion classifies a version string into SRAT runtime
// environments.
func EnvironmentFromVersion(version string) string {
	if version == "0.0.0-dev.0" || strings.Contains(version, "-dev.") {
		return "development"
	}

	if strings.Contains(version, "-rc.") {
		return "prerelease"
	}

	return "production"
}
