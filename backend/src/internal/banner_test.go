package internal

import (
	"bytes"
	"regexp"
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/tlog"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBannerOutputsMetadata(t *testing.T) {
	originalOutput := color.Output
	originalNoColor := color.NoColor
	t.Cleanup(func() {
		color.Output = originalOutput
		color.NoColor = originalNoColor
	})

	var buf bytes.Buffer
	color.Output = &buf
	color.NoColor = true

	originalVersion := config.Version
	originalHash := config.CommitHash
	originalTimestamp := config.BuildTimestamp
	t.Cleanup(func() {
		config.Version = originalVersion
		config.CommitHash = originalHash
		config.BuildTimestamp = originalTimestamp
	})

	config.Version = "1.2.3"
	config.CommitHash = "abcdef"
	config.BuildTimestamp = "2025-09-28T12:00:00"

	previousLevel := tlog.GetLevelString()
	require.NoError(t, tlog.SetLevelFromString("debug"))
	t.Cleanup(func() {
		_ = tlog.SetLevelFromString(previousLevel)
	})

	Banner("SRAT")
	output := stripANSI(buf.String())

	assert.Contains(t, output, "SambaNAS2 Rest Administration Interface")
	assert.Contains(t, output, "Version: 1.2.3")
	assert.Contains(t, output, "abcdef")
	assert.Contains(t, output, "Log level: DEBUG")

	buf.Reset()
	config.BuildTimestamp = ""
	require.NoError(t, tlog.SetLevelFromString("warn"))

	Banner("SRAT")
	output = stripANSI(buf.String())
	assert.Contains(t, output, "Log level: WARN")
}

var ansiRegexp = regexp.MustCompile(`\[[0-9;]*m`)

func stripANSI(input string) string {
	return ansiRegexp.ReplaceAllString(input, "")
}
