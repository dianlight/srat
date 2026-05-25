package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvironmentFromVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{name: "development build", version: "0.0.0-dev.0", expected: "development"},
		{name: "dev suffix", version: "2026.5.0-dev.3", expected: "development"},
		{name: "release candidate", version: "2026.5.0-rc.1", expected: "prerelease"},
		{name: "release", version: "2026.5.0", expected: "production"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, EnvironmentFromVersion(tt.version))
		})
	}
}
