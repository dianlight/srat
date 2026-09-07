//go:build linux

package osutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractMajorMinor(t *testing.T) {
	tests := []struct {
		name          string
		device        string
		expectedMajor int
		expectedMinor int
		expectError   bool
	}{
		{name: "loop zero", device: "/dev/loop0", expectedMajor: 7, expectedMinor: 0},
		{name: "typical value", device: "/dev/loop7", expectedMajor: 7, expectedMinor: 7},
		{name: "multi digit", device: "/dev/loop123", expectedMajor: 7, expectedMinor: 123},
		{name: "max uint32", device: "/dev/loop4294967295", expectedMajor: 7, expectedMinor: 4294967295},
		{name: "overflow uint32", device: "/dev/loop4294967296", expectError: true},
		{name: "huge number truncates without bound check", device: "/dev/loop99999999999", expectError: true},
		{name: "non loop name", device: "/dev/sda1", expectError: true},
		{name: "empty device", device: "", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, err := extractMajorMinor(tt.device)
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMajor, major)
			assert.Equal(t, tt.expectedMinor, minor)
		})
	}
}
